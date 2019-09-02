package main

import (
	"github.com/docopt/docopt-go"
	"github.com/globalsign/mgo"
	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/database"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/karma-go"
)

var (
	version = "[manual build]"
	usage   = "aurorad-rpcd " + version + `

Usage:
  aurorad-rpcd [options]
  aurorad-rpcd -h | --help
  aurorad-rpcd --version

Options:
  -c --config <path>  Configuration file path. [default: /etc/aurorad/rpcd.conf]
  -h --help           Show this screen.
  --version           Show version.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	log.Infof(nil, "starting up aurorad-rpcd %s", version)

	config, err := config.GetRPC(args["--config"].(string))
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

	packages := db.Packages()
	builds := db.Builds()

	err = packages.EnsureIndex(mgo.Index{
		Key:    []string{"name"},
		Unique: true,
	})
	if err != nil {
		log.Fatalf(err, "can't ensure index for collection")
	}

	archives, err := initBus(config.Bus)
	if err != nil {
		log.Fatalf(err, "unable to init bus")
	}

	err = listenAndServe(packages, builds, archives, config)
	if err != nil {
		log.Fatal(err)
	}
}

func initBus(addr string) (bus.Publisher, error) {
	log.Infof(
		karma.Describe("address", addr),
		"connecting to bus",
	)

	conn, err := bus.Dial(addr)
	if err != nil {
		return nil, karma.Format(err, "can't dial bus")
	}

	log.Infof(nil, "connected to bus, creating a channel")

	channel, err := conn.Channel()
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to create bus channel",
		)
	}

	log.Infof(nil, "declaring queue publisher")

	archives, err := channel.GetExchangePublisher(bus.QueueArchives)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to declare exchange publisher",
		)
	}

	log.Infof(nil, "queue publisher %q declared", bus.QueueArchives)

	return archives, nil
}
