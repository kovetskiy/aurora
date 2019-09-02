package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/kovetskiy/aurora/pkg/signature"
	"github.com/kovetskiy/aurora/pkg/storage"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/faces/execution"
	"github.com/reconquest/karma-go"
)

//const (
//    connectionMaxRetries = 10
//    connectionTimeoutMS  = 500
//)

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
	configHistory config.StorageHistory

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

	err := item.Validate()
	if err != nil {
		task.log.Errorf("bug: build is not valid: %s", err)
		return
	}

	task.log.Infof("publishing build: %s", item)

	request := &proto.RequestPushBuild{
		Signature: task.signer.Sign(),
		Build:     item,
	}

	log.Tracef(nil, "%s", log.TraceJSON(request))

	err = task.rpc.Call(
		(*rpc.BuildService).PushBuild,
		request,
		&proto.ResponsePushBuild{},
	)
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
				Error:  err.Error(),
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
				Error:  err.Error(),
			},
		)

		return
	}

	task.push(proto.Build{
		Status: proto.PackageStatusSuccess,
	})
}

func (task *Task) cleanup() error {
	return storage.CleanupRepositoryDirectory(
		task.repoDir,
		task.pkg,
		task.configHistory,
	)
}

func (task *Task) build() (string, error) {
	defer task.teardown()

	var err error

	task.containerName = task.pkg + "-" + fmt.Sprint(time.Now().Unix())

	task.containerID, err = task.runContainer()
	if err != nil {
		return "", err
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

func (task *Task) teardown() {
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

	// close idle connections
	if task.cloud.client != nil {
		task.cloud.client.Close()
	}
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
			task.log.Tracef("%s", strings.TrimRight(data, "\n"))
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

	task.log.Debugf(
		"container %s has been stopped",
		task.containerName,
	)

	err = task.cloud.CopyLogs(task.logsDir, task.containerName, task.pkg)
	if err != nil {
		task.log.Error(
			karma.Format(
				err, "can't write logs for container %s", task.containerName,
			),
		)
	}

	state, err := task.cloud.InspectContainer(container)
	if err != nil {
		return "", karma.Format(err, "unable to inspect container")
	}

	err = state.GetError()
	if err != nil {
		return "", karma.Format(
			err,
			"unexpected container state (maybe old image?)",
		)
	}

	return container, nil
}
