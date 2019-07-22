package main

import (
	"github.com/docopt/docopt-go"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/database"
	"github.com/kovetskiy/aurora/pkg/log"
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

	server := &Server{
		config: config,
		db:     db,
	}

	err = server.Serve()
	if err != nil {
		log.Fatal(err)
	}
}
