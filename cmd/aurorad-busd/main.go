package main

import (
	"github.com/docopt/docopt-go"
	"github.com/globalsign/mgo"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/lorg"
)

var (
	version = "[manual build]"
	usage   = "aurorad-busd " + version + `

Usage:
  aurorad-busd [options]
  aurorad-busd -h | --help
  aurorad-busd --version

Options:
  -c --config <path>  Configuration file path. [default: /etc/aurorad/busd.conf]
  -h --help           Show this screen.
  --version           Show version.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	log.Infof(nil, "starting up aurorad-busd %s", version)

	config, err := config.GetBus(args["--config"].(string))
	if err != nil {
		log.Fatalf(err, "unable to load config")
	}

	if config.Debug {
		log.SetLevel(lorg.LevelDebug)
	}

	if config.Trace {
		log.SetLevel(lorg.LevelTrace)
	}

	packages := db.Packages()

	err = packages.EnsureIndex(mgo.Index{
		Key:    []string{"name"},
		Unique: true,
	})
	if err != nil {
		log.Fatalf(err, "can't ensure index for collection")
	}

	err = listenAndServe(packages, config)
	if err != nil {
		log.Fatal(err)
	}
}
