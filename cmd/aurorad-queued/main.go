package main

import (
	"time"

	"github.com/docopt/docopt-go"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/database"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/lorg"
)

var (
	version = "[manual build]"
	usage   = "aurorad-queued " + version + `

Usage:
  aurorad-queued [options]
  aurorad-queued -h | --help
  aurorad-queued --version

Options:
  -c --config <path>  Configuration file path. [default: /etc/aurorad/queued.conf]
  -h --help           Show this screen.
  --version           Show version.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	log.Infof(nil, "starting up aurorad-queued %s", version)

	config, err := config.GetQueue(args["--config"].(string))
	if err != nil {
		log.Fatalf(err, "unable to load config")
	}

	if config.Debug {
		log.SetLevel(lorg.LevelDebug)
	}

	if config.Trace {
		log.SetLevel(lorg.LevelTrace)
	}

	db, err := database.NewDatabase(config.Database)
	if err != nil {
		log.Fatalf(err, "can't open aurora database")
	}

	bus, err := bus.Dial(config.Bus)
	if err != nil {
		log.Fatalf(err, "can't dial bus")
	}

	channel, err := bus.Channel()
	if err != nil {
		log.Fatalf(err, "can't get bus channel")
	}

	publisher, err := channel.GetQueuePublisher(bus.QueuePackages)
	if err != nil {
		log.Fatalf(err, "can't get queue publisher")
	}

	loopEnqueueBuilds(db.Packages(), publisher, config)
}

func loopEnqueueBuilds(
	pkgs *mgo.Collection,
	publisher bus.Publisher,
	config *config.Queue,
) {
	for {
		pkg := proto.Package{}

		iterator := pkgs.
			Find(bson.M{}).
			Sort("-priority").
			Iter()

		for iterator.Next(&pkg) {
			var interval time.Duration
			var canSkip bool

			switch proto.PackageStatus(pkg.Status) {
			case proto.PackageStatusProcessing:
				interval = config.Interval.Build.StatusProcessing
				canSkip = true

			case proto.PackageStatusSuccess:
				interval = config.Interval.Build.StatusSuccess
				canSkip = true

			case proto.PackageStatusFailure:
				interval = config.Interval.Build.StatusFailure
				canSkip = true
			}

			if canSkip && time.Since(pkg.Date) < interval {
				log.Tracef(
					nil,
					"skip package %s in status %s: "+
						"time since last build %v is less than %v",
					pkg.Name, pkg.Status, time.Since(pkg.Date), interval,
				)

				continue
			}

			log.Debugf(nil, "pushing %s to thread pool queue", pkg.Name)

			err := publisher.Publish(pkg)
			if err != nil {
				log.Fatalf(err, "unable to publish package to queue")
			}
		}

		time.Sleep(config.Interval.Poll)
	}
}
