package main

import (
	"sync"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/database"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/reconquest/karma-go"
)

type Server struct {
	config *config.Queue
	db     *database.Database

	bus     *bus.Connection
	workers *sync.WaitGroup

	queue struct {
		builds bus.Publisher
	}
}

func (server *Server) Serve() error {
	err := server.initBus()
	if err != nil {
		return err
	}

	server.workers = &sync.WaitGroup{}

	server.workers.Add(1)
	go server.enqueueBuilds()

	server.workers.Wait()

	return nil
}

func (server *Server) initBus() error {
	log.Infof(
		karma.Describe("address", server.config.Bus),
		"connecting to bus",
	)

	conn, err := bus.Dial(server.config.Bus)
	if err != nil {
		return karma.Format(err, "can't dial bus")
	}

	log.Infof(nil, "connected to bus, creating a channel")

	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	log.Infof(nil, "declaring queue publisher")

	server.queue.builds, err = channel.GetQueuePublisher(bus.QueueBuilds)
	if err != nil {
		return err
	}

	log.Infof(nil, "queue publisher %q declared", bus.QueueBuilds)

	return nil
}

func (server *Server) enqueueBuilds() {
	defer func() {
		server.workers.Done()
	}()

	for {
		pkg := proto.Package{}

		iterator := server.db.Packages().
			Find(bson.M{}).
			Sort("-priority").
			Iter()

		for iterator.Next(&pkg) {
			var interval time.Duration
			var canSkip bool

			switch proto.PackageStatus(pkg.Status) {
			case proto.PackageStatusProcessing:
				interval = server.config.Interval.Build.StatusProcessing
				canSkip = true

			case proto.PackageStatusSuccess:
				interval = server.config.Interval.Build.StatusSuccess
				canSkip = true

			case proto.PackageStatusFailure:
				interval = server.config.Interval.Build.StatusFailure
				canSkip = true
			}

			if canSkip && time.Since(pkg.UpdatedAt) < interval {
				log.Tracef(
					nil,
					"skip package %s in status %s: "+
						"time since last build %v is less than %v",
					pkg.Name, pkg.Status, time.Since(pkg.UpdatedAt), interval,
				)

				continue
			}

			log.Debugf(nil, "push: %s", pkg.Name)

			err := server.queue.builds.Publish(
				proto.Build{Package: pkg.Name},
			)
			if err != nil {
				log.Fatalf(err, "unable to publish package to queue")
			}
		}

		time.Sleep(server.config.Interval.Poll)
	}
}
