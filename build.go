package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/faces/execution"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
)

const (
	connectionMaxRetries = 10
	connectionTimeoutMS  = 500
)

type build struct {
	collection *mgo.Collection
	pkg        pkg

	instance  string
	repoDir   string
	bufferDir string
	logsDir   string

	cloud *Cloud

	log *lorg.Log

	container string
	ID        string
	process   *execution.Operation
}

var (
	dbLock = &sync.Mutex{}
)

func (build *build) String() string {
	return build.pkg.Name
}

func (build *build) updateStatus(status string) {
	build.pkg.Status = status
	build.pkg.Instance = build.instance

	err := build.collection.Update(
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

	build.pkg.Date = time.Now()
	build.updateStatus("processing")

	archive, err := build.build()
	if err != nil {
		build.log.Error(err)

		build.updateStatus("failure")
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

		build.updateStatus("failure")
		return
	}

	build.log.Infof("adding archive %s to aurora repository", repoPath)

	err = build.repoadd(repoPath)
	if err != nil {
		build.log.Error(
			karma.Format(
				err, "can't update aurora repository",
			),
		)
	}

	build.updateStatus("success")
}

func (build *build) repoadd(path string) error {
	dbLock.Lock()
	defer dbLock.Unlock()

	cmd := exec.Command(
		"repo-add",
		filepath.Join(build.repoDir, "aurora.db.tar"),
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

	build.container = build.pkg.Name + "-" + fmt.Sprint(time.Now().Unix())

	build.ID, err = build.runContainer()
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

func (build *build) runContainer() (string, error) {
	build.log.Debugf("creating container %s", build.container)

	container, err := build.cloud.CreateContainer(
		build.bufferDir,
		build.container,
		build.pkg.Name,
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

	build.log.Debug("building package")

	build.cloud.WaitContainer(container)

	err = build.cloud.WriteLogs(build.logsDir, build.container, build.pkg.Name)
	if err != nil {
		build.log.Error(
			karma.Format(
				err, "can't write logs for container %s", build.container,
			),
		)
	}

	build.log.Debugf(
		"container %s has been stopped",
		build.container,
	)

	return container, nil
}
