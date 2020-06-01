package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/threadpool-go"
)

type Processor struct {
	repoDir   string
	bufferDir string
	logsDir   string
	pool      *threadpool.ThreadPool

	storage *mgo.Collection
	cloud   *Cloud
	config  *Config
	bus     *Bus
}

func NewProcessor(
	storage *mgo.Collection,
	config *Config,
	bus *Bus,
) *Processor {
	return &Processor{
		storage: storage,
		config:  config,
		bus:     bus,
	}
}

func (proc *Processor) Init() error {
	err := proc.removeLock()
	if err != nil {
		return err
	}

	err = cleanupQueue(proc.config.Instance, proc.storage)
	if err != nil {
		return karma.Format(
			err,
			"unable to cleanup queue",
		)
	}

	proc.repoDir, proc.bufferDir, proc.logsDir, err = prepareDirs(proc.config)
	if err != nil {
		return err
	}

	proc.cloud, err = NewCloud(proc.config.BaseImage, proc.config.Resources, proc.config.Threads)
	if err != nil {
		return karma.Format(
			err,
			"unable to init cloud (docker) client",
		)
	}

	err = proc.cloud.Cleanup()
	if err != nil {
		return karma.Format(
			err,
			"unable to cleanup cloud before queue start",
		)
	}

	proc.pool = spawnThreadpool(proc.config.Instance, proc.config.Threads)

	return nil
}

func (proc *Processor) Process() {
	for {
		pkg := proto.Package{}

		iterator := proc.storage.
			Find(bson.M{}).
			Sort("-priority").
			Iter()

		for iterator.Next(&pkg) {
			var since time.Duration
			var interval time.Duration
			var canSkip bool

			since = time.Since(pkg.Date)

			// uh? looks ugly
			switch pkg.Status {
			case proto.BuildStatusProcessing.String():
				interval = proc.config.Interval.Build.StatusProcessing
				canSkip = true

			case proto.BuildStatusSuccess.String():
				interval = proc.config.Interval.Build.StatusSuccess
				canSkip = true

			case proto.BuildStatusFailure.String():
				interval = proc.config.Interval.Build.StatusFailure
				canSkip = true
			}

			if canSkip && since < interval {
				tracef(
					"skip package %s in status %s: "+
						"time since last build %v is less than %v",
					pkg.Name, pkg.Status, since, interval,
				)

				continue
			}

			debugf("pushing %s to thread pool queue", pkg.Name)

			proc.pool.Push(
				&build{
					bus:           proc.bus,
					instance:      proc.config.Instance,
					cloud:         proc.cloud,
					storage:       proc.storage,
					pkg:           pkg,
					repoDir:       proc.repoDir,
					bufferDir:     proc.bufferDir,
					logsDir:       proc.logsDir,
					configHistory: proc.config.History,
				},
			)
		}

		time.Sleep(proc.config.Interval.Poll)
	}
}

func spawnThreadpool(instance string, size int) *threadpool.ThreadPool {
	capacity := size
	if capacity == 0 {
		capacity = runtime.NumCPU()
	}

	pool := threadpool.New()
	pool.Spawn(capacity)

	infof(
		"thread pool with %d threads has been spawned as instance %q",
		capacity, instance,
	)

	return pool
}

func prepareDirs(
	config *Config,
) (repoDir, bufferDir, logsDir string, err error) {
	repoDir, err = filepath.Abs(config.RepoDir)
	if err != nil {
		return "", "", "", err
	}

	bufferDir, err = filepath.Abs(
		filepath.Join(config.BufferDir, config.Instance),
	)
	if err != nil {
		return "", "", "", err
	}

	err = os.RemoveAll(bufferDir)
	if err != nil {
		return "", "", "", karma.Format(
			err,
			"unable to remove buffer directory",
		)
	}

	for _, dir := range []string{
		repoDir,
		bufferDir,
		config.LogsDir,
	} {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", "", "", karma.Format(
				err, "can't mkdir %s", dir,
			)
		}
	}

	return repoDir, bufferDir, config.LogsDir, nil
}

func cleanupQueue(instance string, storage *mgo.Collection) error {
	info, err := storage.UpdateAll(
		bson.M{
			"status":   proto.BuildStatusProcessing,
			"instance": instance,
		},
		bson.M{
			"$set": bson.M{
				"status": proto.BuildStatusUnknown,
			},
		},
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to update old processing items in the queue",
		)
	}

	if info.Updated > 0 {
		infof(
			"%d packages updated from %q to %q",
			info.Updated,
			proto.BuildStatusProcessing,
			proto.BuildStatusUnknown,
		)
	}

	return nil
}

func (proc *Processor) removeLock() error {
	path := filepath.Join(proc.config.RepoDir, packagesDatabaseFile+".lck")

	infof("ensuring database lock file does not exist: %s", path)

	raw, err := ioutil.ReadFile(path)
	if err != nil {
		// That's best case that lck file is not held by something
		if os.IsNotExist(err) {
			return nil
		}

		return karma.Format(
			err,
			"unable to open %s", path,
		)
	}

	warningf("database lock file exists: %s", path)

	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		return karma.
			Describe("path", path).
			Format(
				err,
				"unexpected content in lck file: %q", string(raw),
			)
	}

	warningf("database lock pid: %d", pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		return karma.Format(
			err,
			"unable to find process %d", pid,
		)
	}

	defer process.Release()

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		// process found, we can't remove lock
		return fmt.Errorf("process %d that locked %s is still running", pid, path)
	}

	warningf("database lock process is not running: %d", pid)

	err = os.Remove(path)
	if err != nil {
		return karma.Format(
			err,
			"unable to remove lck file: %s",
			path,
		)
	}

	warningf("database lock file has been removed: %s", path)

	return nil
}
