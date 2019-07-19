package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/faces/execution"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
	"github.com/reconquest/regexputil-go"
)

const (
	reArchiveTime = `(?P<time>\d+)`
	reArchiveName = `(?P<name>[a-z0-9][a-z0-9@\._+-]+)`
	reArchiveVer  = `(?P<ver>[a-z0-9_.]+-[0-9]+)`
	reArchiveArch = `(?P<arch>(i686|x86_64))`
	reArchiveExt  = `(?P<ext>tar(.(gz|bz2|xz|lrz|lzo|sz))?)`

	packagesDatabaseFile = "aurora.db.tar"
)

var (
	reArchiveFilename = regexp.MustCompile(`^` + reArchiveTime +
		`\.` + reArchiveName +
		`-` + reArchiveVer +
		`-` + reArchiveArch +
		`\.pkg\.` + reArchiveExt + `$`)
)

const (
	connectionMaxRetries = 10
	connectionTimeoutMS  = 500
)

type build struct {
	log *lorg.Log
	pkg proto.Package

	instance      string
	repoDir       string
	bufferDir     string
	logsDir       string
	configHistory config.History

	cloud         *Cloud
	containerName string
	containerID   string
	process       *execution.Operation

	queue struct {
		builds   bus.Publisher
		packages bus.Consumer
		logs     bus.Publisher
	}
}

var (
	dbLock = &sync.Mutex{}
)

func (build *build) String() string {
	return build.pkg.Name
}

func (build *build) publishBuild(item proto.Build) {
	build.log.Infof("status: %s", item.Status)

	item.Package = build.pkg.Name
	item.At = time.Now()
	item.Instance = build.instance

	build.log.Infof("publishing build: %s", item)

	err := build.queue.builds.Publish(item)
	if err != nil {
		build.log.Error(
			karma.Format(
				err, "can't push build status",
			),
		)
		return
	}
}

func (build *build) init() bool {
	build.log = log.Logger.NewChildWithPrefix(
		fmt.Sprintf("(%s)", build.pkg.Name),
	)

	return true
}

func (build *build) Process() {
	if !build.init() {
		return
	}

	build.cleanup()

	build.publishBuild(proto.Build{Status: proto.PackageStatusProcessing})

	archive, err := build.build()
	if err != nil {
		build.log.Error(err)

		build.publishBuild(
			proto.Build{
				Status: proto.PackageStatusFailure,
				Error:  err,
			},
		)
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

		build.publishBuild(
			proto.Build{
				Status: proto.PackageStatusFailure,
				Error:  err,
			},
		)
		return
	}

	build.publishBuild(proto.Build{
		Status: proto.PackageStatusSuccess,
	})
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
	for version, _ := range builds {
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

func (build *build) build() (string, error) {
	defer build.shutdown()

	var err error

	build.containerName = build.pkg.Name + "-" + fmt.Sprint(time.Now().Unix())

	build.containerID, err = build.runContainer()
	if err != nil {
		return "", karma.Format(
			err, "can't run container for building package",
		)
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
	if build.containerID != "" {
		err := build.cloud.DestroyContainer(build.containerID)
		if err != nil {
			build.log.Error(
				karma.Format(
					err, "can't destroy container %s", build.containerID,
				),
			)
		}

		build.log.Debugf("container %s has been destroyed", build.containerName)
	}

	build.cloud.client.Close()
}

func (build *build) runContainer() (string, error) {
	build.log.Debugf("creating container %s", build.containerName)

	container, err := build.cloud.CreateContainer(
		build.bufferDir,
		build.containerName,
		build.pkg.Name,
	)
	if err != nil {
		return "", karma.Format(
			err, "can't create container",
		)
	}
	build.log.Debugf(
		"container %s has been created",
		build.containerName,
	)

	err = build.cloud.StartContainer(container)
	if err != nil {
		return "", karma.Format(
			err, "can't start container",
		)
	}

	build.log.Debug("building package")

	routines := &sync.WaitGroup{}
	routines.Add(1)
	go func() {
		defer routines.Done()
		build.cloud.WaitContainer(container)
	}()

	routines.Add(1)
	go func() {
		defer routines.Done()
		build.cloud.FollowLogs(container, func(data string) {
			err := build.queue.logs.Publish(proto.BuildLogChunk{
				Package: build.pkg.Name,
				Data:    data,
			})
			if err != nil {
				build.log.Errorf("unable to publish log: %s", err)
			}
		})
	}()

	routines.Wait()

	err = build.cloud.WriteLogs(build.logsDir, build.containerName, build.pkg.Name)
	if err != nil {
		build.log.Error(
			karma.Format(
				err, "can't write logs for container %s", build.containerName,
			),
		)
	}

	build.log.Debugf(
		"container %s has been stopped",
		build.containerName,
	)

	return container, nil
}
