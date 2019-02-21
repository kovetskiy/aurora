package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/faces/execution"
	"github.com/reconquest/lexec-go"
	"github.com/reconquest/ser-go"
	"github.com/theairkit/runcmd"
)

const (
	connectionMaxRetries = 10
	connectionTimeoutMS  = 500
)

type build struct {
	database *database
	pkg      pkg

	repositoryDir string
	buildsDir     string
	logsDir       string

	cloud Cloud

	log *lorg.Log

	container string
	address   string
	dir       string
	ID        string
	process   *execution.Operation

	session runcmd.Runner
	done    map[string]bool
}

var (
	dbLock = &sync.Mutex{}
)

func (build *build) String() string {
	return build.pkg.Name
}

func (build *build) updateStatus(status string) {
	build.pkg.Status = status

	build.database.set(build.pkg.Name, build.pkg)

	err := saveDatabase(build.database)
	if err != nil {
		build.log.Error(
			ser.Errorf(
				err, "can't save database with new package status",
			),
		)
		return
	}

	build.log.Infof("status: %s", status)
}

func (build *build) init() bool {
	var err error

	build.log = logger.NewChildWithPrefix(
		fmt.Sprintf("(%s)", build.pkg.Name),
	)

	build.cloud = Cloud{}
	build.cloud.client, err = client.NewEnvClient()
	if err != nil {
		build.log.Error(err)
		return false
	}

	build.dir = "/build"

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

	build.log.Infof("package has been built: %s", archive)
	build.log.Infof("adding archive %s to aurora repository", archive)

	err = build.repoadd(archive)
	if err != nil {
		build.log.Error(
			ser.Errorf(
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
		filepath.Join(build.repositoryDir, "aurora.db.tar"),
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
		return "", ser.Errorf(
			err, "can't run container for building package",
		)
	}

	archives, err := filepath.Glob(
		filepath.Join(
			fmt.Sprintf("%s/%s*.pkg.*", build.repositoryDir, build.pkg.Name),
		),
	)
	if err != nil {
		return "", ser.Errorf(
			err, "can't stat built package archive",
		)
	}

	for _, archive := range archives {
		target := archive
		return target, nil
	}

	return "", errors.New("built archive file not found")
}

func (build *build) shutdown() {
	if build.ID != "" {
		err := build.cloud.DestroyContainer(build.ID)
		if err != nil {
			build.log.Error(
				ser.Errorf(
					err, "can't destroy container %s", build.ID,
				),
			)
		}

		build.log.Debugf("container %s has been destroyed", build.container)
	}
}

func (build *build) runContainer() (string, error) {
	build.log.Debugf("creating container %s", build.container)

	container, err := build.cloud.CreateContainer(
		build.repositoryDir, build.container, build.pkg.Name,
	)
	if err != nil {
		return "", ser.Errorf(
			err, "can't create container",
		)
	}
	build.log.Debugf(
		"container %s has been created",
		build.container,
	)

	err = build.cloud.StartContainer(container)
	if err != nil {
		return "", ser.Errorf(
			err, "can't start container",
		)
	}

	build.log.Debug("building package")

	build.cloud.WaitContainer(container)

	err = build.cloud.WriteLogs(build.logsDir, build.container, build.pkg.Name)

	if err != nil {
		build.log.Error(
			ser.Errorf(
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
