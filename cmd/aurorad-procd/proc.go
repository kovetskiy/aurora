package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/kovetskiy/aurora/pkg/signature"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/threadpool-go"
)

type Processor struct {
	repoDir   string // prepared dirs
	bufferDir string
	logsDir   string

	pool *threadpool.ThreadPool

	cloud  *Cloud
	config *config.Proc

	rpc    *rpc.Client
	signer *signature.Signer

	bus   *bus.Connection
	queue struct {
		builds bus.Consumer
	}
}

func NewProcessor(
	config *config.Proc,
) (*Processor, error) {
	var err error
	proc := &Processor{
		config: config,
	}

	proc.repoDir, proc.bufferDir, proc.logsDir, err = prepareDirs(proc.config)
	if err != nil {
		return nil, err
	}

	proc.cloud, err = NewCloud(proc.config.BaseImage)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to init cloud (docker) client",
		)
	}

	err = proc.cloud.Cleanup()
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to cleanup cloud before queue start",
		)
	}

	err = proc.initBus()
	if err != nil {
		return nil, err
	}

	proc.signer, err = signature.NewSigner(proc.config.Key)
	if err != nil {
		return nil, err
	}

	proc.rpc = rpc.NewClient(proc.config.RPC)
	proc.pool = spawnThreadpool(proc.config.Instance, proc.config.Threads)

	return proc, nil
}

func (proc *Processor) initBus() error {
	var err error
	proc.bus, err = bus.Dial(proc.config.Bus)
	if err != nil {
		return karma.Format(
			err,
			"unable to dial to bus",
		)
	}

	channel, err := proc.bus.Channel()
	if err != nil {
		return err
	}

	proc.queue.builds, err = channel.GetQueueConsumer(bus.QueueBuilds)
	if err != nil {
		return err
	}

	return nil
}

func (proc *Processor) LoopServe() {
	for {
		delivery, ok := proc.queue.builds.Consume()
		if !ok {
			log.Info("queue builds has been closed, stopping")
			break
		}

		var build proto.Build
		err := delivery.Decode(&build)
		if err != nil {
			log.Errorf(
				err,
				"got unexpected item in queue: %q",
				delivery.GetBody(),
			)

			continue
		}

		task := &Task{
			pkg:           build.Package,
			instance:      proc.config.Instance,
			cloud:         proc.cloud,
			repoDir:       proc.repoDir,
			bufferDir:     proc.bufferDir,
			logsDir:       proc.logsDir,
			configHistory: proc.config.History,
			rpc:           proc.rpc,
			signer:        proc.signer,
		}

		log.Tracef(nil, "pushing to threadpool: %s", build.Package)

		proc.pool.Push(task)
	}
}

func spawnThreadpool(instance string, size int) *threadpool.ThreadPool {
	capacity := size
	if capacity == 0 {
		capacity = runtime.NumCPU()
	}

	pool := threadpool.New()
	pool.Spawn(capacity)

	log.Infof(
		nil,
		"thread pool with %d threads has been spawned as instance %q",
		capacity, instance,
	)

	return pool
}

func prepareDirs(
	config *config.Proc,
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

// need to move to queued
// func cleanupQueue(instance string, storage *mgo.Collection) error {
//     info, err := storage.UpdateAll(
//         bson.M{
//             "status":   proto.PackageStatusProcessing,
//             "instance": instance,
//         },
//         bson.M{
//             "$set": bson.M{
//                 "status": proto.PackageStatusUnknown,
//             },
//         },
//     )
//     if err != nil {
//         return karma.Format(
//             err,
//             "unable to update old processing items in the queue",
//         )
//     }

//     if info.Updated > 0 {
//         log.Infof(
//             nil,
//             "%d packages updated from %q to %q",
//             info.Updated,
//             proto.PackageStatusProcessing,
//             proto.PackageStatusUnknown,
//         )
//     }

//     return nil
// }
