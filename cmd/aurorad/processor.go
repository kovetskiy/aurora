package main

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/aurora"
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
	err := cleanupQueue(proc.config.Instance, proc.storage)
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

	proc.cloud, err = NewCloud(proc.config.BaseImage)
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
		pkg := aurora.Package{}

		iterator := proc.storage.
			Find(bson.M{}).
			Sort("-priority").
			Iter()

		for iterator.Next(&pkg) {
			var since time.Duration
			var interval time.Duration
			var canSkip bool

			since = time.Since(pkg.Date)
			switch pkg.Status {
			case BuildStatusProcessing.String():
				interval = proc.config.Interval.Build.StatusProcessing
				canSkip = true

			case BuildStatusSuccess.String():
				interval = proc.config.Interval.Build.StatusSuccess
				canSkip = true

			case BuildStatusFailure.String():
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
			"status":   BuildStatusProcessing,
			"instance": instance,
		},
		bson.M{
			"$set": bson.M{
				"status": BuildStatusUnknown,
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
			BuildStatusProcessing,
			BuildStatusUnknown,
		)
	}

	return nil
}
