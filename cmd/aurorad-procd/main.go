package main

import (
	"github.com/docopt/docopt-go"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/lorg"
)

var (
	version = "[manual build]"
	usage   = "aurorad-procd " + version + `

Usage:
  aurorad-procd [options]
  aurorad-procd -h | --help
  aurorad-procd --version

Options:
  -c --config <path>  Configuration file path. [default: /etc/aurorad/procd.conf]
  -h --help           Show this screen.
  --version           Show version.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	log.Infof(nil, "starting up aurorad-procd %s", version)

	config, err := config.GetProc(args["--config"].(string))
	if err != nil {
		log.Fatalf(err, "unable to load config")
	}

	if config.Debug {
		log.SetLevel(lorg.LevelDebug)
	}

	if config.Trace {
		log.SetLevel(lorg.LevelTrace)
	}

	proc, err := NewProcessor(config)
	if err != nil {
		log.Fatal(err)
	}

	proc.Process()
}
