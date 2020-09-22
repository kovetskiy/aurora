package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/faces/execution"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
	"github.com/reconquest/regexputil-go"
)

const (
	FAILURES_TO_REMOVE = 3
)

var ErrPkgverNotChanged = errors.New("pkgver not changed")

type execWriter struct {
	logger  lorg.Logger
	publish func(string)
}

func (writer *execWriter) Write(buffer []byte) (int, error) {
	writer.logger.Trace(string(buffer))
	if writer.publish != nil {
		writer.publish(strings.TrimRight(string(buffer), "\n \t") + "\n")
	}
	return len(buffer), nil
}

const (
	reArchiveTime = `(?P<time>\d+)`
	reArchiveName = `(?P<name>[a-zA-Z0-9][a-zA-Z0-9@\._+-]+)`
	reArchiveVer  = `(?P<ver>[a-zA-Z0-9_.:]+-[0-9]+)`
	reArchiveArch = `(?P<arch>(i686|x86_64|any))`
	reArchiveExt  = `(?P<ext>tar(.(gz|bz2|xz|zst|lrz|lzo|sz))?)`

	packagesDatabaseFile = "aurora.db.tar"
)

var reArchiveFilename = regexp.MustCompile(
	`^` + reArchiveTime +
		`\.` + reArchiveName +
		`-` + reArchiveVer +
		`-` + reArchiveArch +
		`\.pkg\.` + reArchiveExt + `$`,
)

const (
	connectionMaxRetries = 10
	connectionTimeoutMS  = 500
)

type build struct {
	storage *mgo.Collection
	pkg     proto.Package

	instance      string
	repoDir       string
	bufferDir     string
	logsDir       string
	configHistory ConfigHistory

	cloud *Cloud

	log *lorg.Log

	container string
	ID        string
	process   *execution.Operation
	bus       *Bus
}

var dbLock = &sync.Mutex{}

func (build *build) String() string {
	return build.pkg.Name
}

func (build *build) removePackage() {
	err := build.storage.Remove(bson.M{"name": build.pkg.Name})
	if err != nil {
		build.log.Error(
			karma.Format(
				err, "unable to remove package",
			),
		)
	}
}

func (build *build) updateStatus(status proto.BuildStatus) {
	build.pkg.Status = status.String()
	build.pkg.Instance = build.instance

	build.bus.Publish(build.pkg.Name, status)

	err := build.storage.Update(
		bson.M{"name": build.pkg.Name},
		build.pkg,
	)
	if err != nil {
		build.log.Error(
			karma.Format(
				err, "can't update new package status",
			),
		)
		return
	}

	build.log.Infof("status: %s", status)
}

func (build *build) init() bool {
	build.log = logger.NewChildWithPrefix(
		fmt.Sprintf("(%s)", build.pkg.Name),
	)

	return true
}

func (build *build) Process() {
	if !build.init() {
		return
	}

	build.cleanup()

	oldstatus := build.pkg.Status

	build.pkg.Date = time.Now()
	build.updateStatus(proto.BuildStatusProcessing)

	archive, err := build.build(oldstatus)
	if err != nil {
		if err == ErrPkgverNotChanged {
			build.log.Infof("pkgver not changed, skipping; pkgver=%v", build.pkg.Version)
			build.updateStatus(proto.BuildStatusSuccess)
			return
		}

		build.log.Error(err)

		build.pkg.Failures++
		build.updateStatus(proto.BuildStatusFailure)

		if build.pkg.Failures >= FAILURES_TO_REMOVE && build.pkg.Priority == 0 {
			build.log.Warningf(
				"package failed %d times, removing it from the database",
				build.pkg.Failures,
			)
			build.removePackage()
		}

		return
	}

	build.log.Infof("package is ready in buffer: %s", archive)

	repoPath := filepath.Join(build.repoDir, filepath.Base(archive))

	err = os.Rename(archive, repoPath)
	if err != nil {
		build.log.Error(
			karma.Format(
				err,
				"unable to move file from buffer",
			),
		)

		build.updateStatus(proto.BuildStatusFailure)
		return
	}

	build.log.Infof("adding archive %s to aurora repository", repoPath)

	err = build.repoAdd(repoPath)
	if err != nil {
		build.log.Error(
			karma.Format(
				err, "can't update aurora repository",
			),
		)
		build.updateStatus(proto.BuildStatusFailure)

		return
	}

	build.pkg.Failures = 0
	build.updateStatus(proto.BuildStatusSuccess)
}

func (build *build) cleanup() error {
	globbed, err := filepath.Glob(
		filepath.Join(
			fmt.Sprintf("%s/*.%s-*-*-*.pkg.*", build.repoDir, build.pkg.Name),
		),
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to glob for packages",
		)
	}

	type archive struct {
		Time     string
		Basename string
	}

	builds := map[string][]archive{}
	for _, fullpath := range globbed {
		basename := filepath.Base(fullpath)

		matches := reArchiveFilename.FindStringSubmatch(basename)

		name := regexputil.Subexp(reArchiveFilename, matches, "name")
		if name != build.pkg.Name {
			continue
		}

		ver := regexputil.Subexp(reArchiveFilename, matches, "ver")
		time := regexputil.Subexp(reArchiveFilename, matches, "time")

		builds[ver] = append(builds[ver], archive{
			Time:     time,
			Basename: basename,
		})
	}

	versions := []string{}
	for version := range builds {
		versions = append(versions, version)
	}

	trash := []string{}
	if len(versions) > build.configHistory.Versions {
		max := build.configHistory.Versions

		sort.Sort(sort.StringSlice(versions))

		for _, version := range versions[max:] {
			for _, archive := range builds[version] {
				trash = append(trash, archive.Basename)
			}

			delete(builds, version)
		}
	}

	for _, archives := range builds {
		if len(archives) <= build.configHistory.BuildsPerVersion {
			continue
		}

		sort.Slice(archives, func(i, j int) bool {
			return archives[i].Time < archives[j].Time
		})

		for _, archive := range archives[build.configHistory.BuildsPerVersion:] {
			trash = append(trash, archive.Basename)
		}
	}

	for _, archive := range trash {
		fullpath := filepath.Join(build.repoDir, archive)

		build.log.Tracef("removing old pkg: %s", fullpath)

		err := os.Remove(fullpath)
		if err != nil {
			build.log.Error(
				karma.Format(
					err,
					"unable to remove old pkg: %s",
					fullpath,
				),
			)
		}
	}

	return nil
}

func (build *build) repoAdd(path string) error {
	dbLock.Lock()
	defer dbLock.Unlock()

	cmd := exec.Command(
		"repo-add",
		filepath.Join(build.repoDir, packagesDatabaseFile),
		path,
	)

	err := lexec.NewExec(lexec.Loggerf(build.log.Tracef), cmd).Run()
	if err != nil {
		return err
	}

	return nil
}

func (build *build) repoRemove(path string) error {
	dbLock.Lock()
	defer dbLock.Unlock()

	cmd := exec.Command(
		"repo-remove",
		filepath.Join(build.repoDir, packagesDatabaseFile),
		path,
	)

	err := lexec.NewExec(lexec.Loggerf(build.log.Tracef), cmd).Run()
	if err != nil {
		return err
	}

	return nil
}

func (build *build) build(oldstatus string) (string, error) {
	defer build.shutdown()

	var err error

	build.container = build.pkg.Name + "-" + fmt.Sprint(time.Now().Unix())

	build.ID, err = build.start(oldstatus)
	if err != nil {
		return "", err
	}

	archives, err := filepath.Glob(
		filepath.Join(
			fmt.Sprintf("%s/%s/*.pkg.*", build.bufferDir, build.pkg.Name),
		),
	)
	if err != nil {
		return "", karma.Format(
			err, "can't stat built package archive",
		)
	}

	if len(archives) > 0 {
		target := archives[0]

		stat, err := os.Stat(target)
		if err != nil {
			return "", err
		}

		newest := stat.ModTime()

		for _, archive := range archives {
			stat, err = os.Stat(archive)
			if err != nil {
				return "", err
			}

			if stat.ModTime().After(newest) {
				target = archive
				newest = stat.ModTime()
			}
		}

		return target, nil
	}

	return "", errors.New("built archive file not found")
}

func (build *build) shutdown() {
	if build.ID != "" {
		err := build.cloud.DestroyContainer(build.ID)
		if err != nil {
			build.log.Error(
				karma.Format(
					err, "can't destroy container %s", build.ID,
				),
			)
		}

		build.log.Debugf("container %s has been destroyed", build.container)
	}

	build.cloud.client.Close()
}

func (build *build) start(oldstatus string) (string, error) {
	build.log.Debugf("creating container %s", build.container)

	build.bus.Publish(build.pkg.Name, "builder: Creating container for makepkg\n")

	container, err := build.cloud.CreateContainer(
		build.bufferDir,
		build.container,
		build.pkg.Name,
		build.pkg.CloneURL,
		build.pkg.Subdir,
	)
	if err != nil {
		return "", karma.Format(
			err, "can't create container",
		)
	}

	build.log.Debugf(
		"container %s has been created",
		build.container,
	)

	err = build.cloud.StartContainer(container)
	if err != nil {
		return "", karma.Format(
			err, "can't start container",
		)
	}

	build.bus.Publish(build.pkg.Name, "builder: Retrieving PKGVER\n")

	pkgverAt := time.Now()
	pkgver, err := build.getVersion(container)
	build.pkg.PkgverTime = time.Since(pkgverAt)

	if err != nil {
		return "", karma.Format(
			err,
			"unable to get pkgver",
		)
	}

	if build.pkg.Version == pkgver && oldstatus != proto.BuildStatusFailure.String() {
		build.bus.Publish(build.pkg.Name, "Builder: PKGVER is not changed")
		return "", ErrPkgverNotChanged
	}

	build.bus.Publish(
		build.pkg.Name,
		fmt.Sprintf("Builder: PKGVER is %q (was %q)\n", pkgver, build.pkg.Version),
	)

	build.log.Debug("building package")
	build.bus.Publish(build.pkg.Name, "builder: Starting build\n")

	runAt := time.Now()
	_, err = build.WaitRun(container)
	build.pkg.BuildTime = time.Since(runAt)

	if err != nil {
		return "", err
	}

	build.pkg.Version = pkgver

	logErr := build.cloud.WriteLogs(build.logsDir, build.container, build.pkg.Name)
	if logErr != nil {
		build.log.Error(
			karma.Format(
				logErr, "can't write logs for container %s", build.container,
			),
		)
	}

	build.bus.Publish(build.pkg.Name, "builder: Build finished\n")

	build.log.Debugf(
		"container %s has been stopped",
		build.container,
	)

	return container, err
}

func (build *build) getVersion(container string) (string, error) {
	err := build.cloud.Exec(
		context.Background(), build.log, func(log string) {
			build.bus.Publish(build.pkg.Name, "pkgver: "+log)
		}, container, []string{"/app/pkgver.sh"}, nil,
	)
	if err != nil {
		return "", karma.Format(err, "pkgver.sh failed")
	}

	path := fmt.Sprintf("%s/%s/pkgver", build.bufferDir, build.pkg.Name)
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", karma.Format(
			err,
			"unable to read file after pkgver: %s", path,
		)
	}

	err = os.Remove(path)
	if err != nil {
		return "", karma.Format(
			err,
			"unable to remove pkgver file",
		)
	}

	return string(contents), nil
}

func (build *build) run(ctx context.Context, container string) error {
	err := build.cloud.Exec(
		ctx, build.log, func(log string) {
			build.bus.Publish(build.pkg.Name, "makepkg: "+log)
		},
		container, []string{"/app/run.sh"}, nil,
	)
	if err != nil {
		return karma.Format(err, "run.sh failed")
	}

	return nil
}

func (build *build) WaitRun(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
	defer cancel()

	result := make(chan error, 1)
	routines := &sync.WaitGroup{}
	routines.Add(1)
	go func() {
		defer routines.Done()

		result <- build.run(ctx, name)
	}()

	routines.Wait()

	select {
	case err := <-result:
		if err != nil {
			return false, err
		}
		return true, nil
	case <-ctx.Done():
		cancel()
		return false, <-result
	}
}
