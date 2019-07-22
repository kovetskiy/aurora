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

	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/kovetskiy/aurora/pkg/signature"
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

type Task struct {
	inited bool
	log    *lorg.Log
	pkg    string
	rpc    *rpc.Client
	signer *signature.Signer

	instance      string
	repoDir       string
	bufferDir     string
	logsDir       string
	configHistory config.History

	cloud         *Cloud
	containerName string
	containerID   string
	process       *execution.Operation
}

var (
	dbLock = &sync.Mutex{}
)

func (task *Task) String() string {
	return task.pkg
}

func (task *Task) push(item proto.Build) {
	task.log.Infof("status: %s", item.Status)

	item.Package = task.pkg
	item.At = time.Now()

	task.log.Infof("publishing build: %s", item)

	err := task.rpc.Call((*rpc.BuildService).PushBuild, &proto.RequestPushBuild{
		Signature: task.signer.Sign(),
		Build:     item,
	}, &proto.ResponsePushBuild{})
	if err != nil {
		task.log.Error(
			karma.Format(
				err, "can't push build status",
			),
		)
		return
	}
}

func (task *Task) init() bool {
	if task.inited {
		return true
	}

	task.log = log.Logger.NewChildWithPrefix(
		fmt.Sprintf("(%s)", task.pkg),
	)

	return true
}

func (task *Task) Process() {
	if !task.init() {
		return
	}

	task.cleanup()

	task.push(proto.Build{Status: proto.PackageStatusProcessing})

	archive, err := task.build()
	if err != nil {
		task.log.Error(err)

		task.push(
			proto.Build{
				Status: proto.PackageStatusFailure,
				Error:  err,
			},
		)
		return
	}

	task.log.Infof("package is ready in buffer: %s", archive)

	repoPath := filepath.Join(task.repoDir, filepath.Base(archive))

	err = os.Rename(archive, repoPath)
	if err != nil {
		task.log.Error(
			karma.Format(
				err,
				"unable to move file from buffer",
			),
		)

		task.push(
			proto.Build{
				Status: proto.PackageStatusFailure,
				Error:  err,
			},
		)
		return
	}

	task.push(proto.Build{
		Status: proto.PackageStatusSuccess,
	})
}

func (task *Task) cleanup() error {
	globbed, err := filepath.Glob(
		filepath.Join(
			fmt.Sprintf("%s/*.%s-*-*-*.pkg.*", task.repoDir, task.pkg),
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
		if name != task.pkg {
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
	if len(versions) > task.configHistory.Versions {
		max := task.configHistory.Versions

		sort.Sort(sort.StringSlice(versions))

		for _, version := range versions[max:] {
			for _, archive := range builds[version] {
				trash = append(trash, archive.Basename)
			}

			delete(builds, version)
		}
	}

	for _, archives := range builds {
		if len(archives) <= task.configHistory.BuildsPerVersion {
			continue
		}

		sort.Slice(archives, func(i, j int) bool {
			return archives[i].Time < archives[j].Time
		})

		for _, archive := range archives[task.configHistory.BuildsPerVersion:] {
			trash = append(trash, archive.Basename)
		}
	}

	for _, archive := range trash {
		fullpath := filepath.Join(task.repoDir, archive)

		task.log.Tracef("removing old pkg: %s", fullpath)

		err := os.Remove(fullpath)
		if err != nil {
			task.log.Error(
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

func (task *Task) repoRemove(path string) error {
	dbLock.Lock()
	defer dbLock.Unlock()

	cmd := exec.Command(
		"repo-remove",
		filepath.Join(task.repoDir, packagesDatabaseFile),
		path,
	)

	err := lexec.NewExec(lexec.Loggerf(task.log.Tracef), cmd).Run()
	if err != nil {
		return err
	}

	return nil
}

func (task *Task) build() (string, error) {
	defer task.shutdown()

	var err error

	task.containerName = task.pkg + "-" + fmt.Sprint(time.Now().Unix())

	task.containerID, err = task.runContainer()
	if err != nil {
		return "", karma.Format(
			err, "can't run container for building package",
		)
	}

	archives, err := filepath.Glob(
		filepath.Join(
			fmt.Sprintf("%s/%s/*.pkg.*", task.bufferDir, task.pkg),
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

func (task *Task) shutdown() {
	if task.containerID != "" {
		err := task.cloud.DestroyContainer(task.containerID)
		if err != nil {
			task.log.Error(
				karma.Format(
					err, "can't destroy container %s", task.containerID,
				),
			)
		}

		task.log.Debugf("container %s has been destroyed", task.containerName)
	}

	task.cloud.client.Close()
}

func (task *Task) runContainer() (string, error) {
	task.log.Debugf("creating container %s", task.containerName)

	container, err := task.cloud.CreateContainer(
		task.bufferDir,
		task.containerName,
		task.pkg,
	)
	if err != nil {
		return "", karma.Format(
			err, "can't create container",
		)
	}
	task.log.Debugf(
		"container %s has been created",
		task.containerName,
	)

	err = task.cloud.StartContainer(container)
	if err != nil {
		return "", karma.Format(
			err, "can't start container",
		)
	}

	task.log.Debug("building package")

	routines := &sync.WaitGroup{}
	routines.Add(1)
	go func() {
		defer routines.Done()
		task.cloud.WaitContainer(container)
	}()

	routines.Add(1)
	go func() {
		defer routines.Done()
		task.cloud.FollowLogs(container, func(data string) {
			//err := task.queue.logs.Publish(proto.BuildLogChunk{
			//    Package: task.pkg,
			//    Data:    data,
			//})
			//if err != nil {
			//    task.log.Errorf("unable to publish log: %s", err)
			//}
		})
	}()

	routines.Wait()

	err = task.cloud.WriteLogs(task.logsDir, task.containerName, task.pkg)
	if err != nil {
		task.log.Error(
			karma.Format(
				err, "can't write logs for container %s", task.containerName,
			),
		)
	}

	task.log.Debugf(
		"container %s has been stopped",
		task.containerName,
	)

	return container, nil
}
