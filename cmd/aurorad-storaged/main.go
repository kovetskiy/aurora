package main

import (
	"github.com/docopt/docopt-go"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/lorg"
)

var (
	version = "[manual build]"
	usage   = "aurorad-storaged " + version + `

Usage:
  aurorad-storaged [options]
  aurorad-storaged -h | --help
  aurorad-storaged --version

Options:
  -c --config <path>  Configuration file path. [default: /etc/aurorad/storaged.conf]
  -h --help           Show this screen.
  --version           Show version.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	log.Infof(nil, "starting up aurorad-storaged %s", version)

	config, err := config.GetStorage(args["--config"].(string))
	if err != nil {
		log.Fatalf(err, "unable to load config")
	}

	if config.Debug {
		log.SetLevel(lorg.LevelDebug)
	}

	if config.Trace {
		log.SetLevel(lorg.LevelTrace)
	}

	server := &Server{
		config: config,
	}

	err = server.Serve()
	if err != nil {
		log.Fatal(err)
	}
}
