package main

import (
	"github.com/docopt/docopt-go"
	"github.com/globalsign/mgo"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/database"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/lorg"
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

	err = listenAndServe(packages, builds, config)
	if err != nil {
		log.Fatal(err)
	}
}
