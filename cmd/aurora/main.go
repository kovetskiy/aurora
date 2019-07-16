package main

import "github.com/docopt/docopt-go"
import "log"

var (
	version = "[manual build]"
	usage   = "aurora " + version + `

Aurora is a client for aurorad.

Usage:
  aurora [options] -Q
  aurora -h | --help
  aurora --version

Options:
  -Q --query          Make a query request.
  -s --server <addr>  Address of aurorad server. [default: aur.reconquest.io]
  -h --help           Show this screen.
  --version           Show version.
`
)

type (
	Options struct {
		Server string
		Query  bool
	}
)

func main() {
	args, err := docopt.ParseArgs(usage, nil, version)
	if err != nil {
		panic(err)
	}

	var opts Options
	err = args.Bind(&opts)
	if err != nil {
		panic(err)
	}

	switch {
	case opts.Query:
		err = handleQuery(opts)
	}

	if err != nil {
		log.Fatalln(err)
	}
}
