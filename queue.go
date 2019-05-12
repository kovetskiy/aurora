package main

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/threadpool-go"
)

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

func cleanupQueue(instance string, collection *mgo.Collection) error {
	info, err := collection.UpdateAll(
		bson.M{
			"status":   StatusProcessing,
			"instance": instance,
		},
		bson.M{
			"$set": bson.M{
				"status": StatusUnknown,
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
			StatusProcessing,
			StatusUnknown,
		)
	}

	return nil
}

func processQueue(collection *mgo.Collection, config *Config) error {
	pool := spawnThreadpool(config.Instance, config.Threads)

	repoDir, bufferDir, logsDir, err := prepareDirs(config)
	if err != nil {
		return err
	}

	err = cleanupQueue(config.Instance, collection)
	if err != nil {
		return karma.Format(
			err,
			"unable to cleanup queue",
		)
	}

	cloud, err := NewCloud(config.BaseImage)
	if err != nil {
		return karma.Format(
			err,
			"unable to init cloud (docker) client",
		)
	}

	err = cloud.Cleanup()
	if err != nil {
		return karma.Format(
			err,
			"unable to cleanup cloud before queue start",
		)
	}

	for {
		pkg := Package{}
		packages := collection.Find(bson.M{}).Iter()

		for packages.Next(&pkg) {
			var since time.Duration
			var interval time.Duration
			var canSkip bool

			since = time.Since(pkg.Date)
			switch pkg.Status {
			case StatusProcessing:
				interval = config.Interval.Build.StatusProcessing
				canSkip = true

			case StatusSuccess:
				interval = config.Interval.Build.StatusSuccess
				canSkip = true

			case StatusFailure:
				interval = config.Interval.Build.StatusFailure
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

			pool.Push(
				&build{
					instance:      config.Instance,
					cloud:         cloud,
					collection:    collection,
					pkg:           pkg,
					repoDir:       repoDir,
					bufferDir:     bufferDir,
					logsDir:       logsDir,
					configHistory: config.History,
				},
			)
		}

		time.Sleep(config.Interval.Poll)
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
