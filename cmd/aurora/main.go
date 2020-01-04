package main

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/reconquest/karma-go"
)

var (
	version = "[manual build]"
	usage   = "aurora " + version + os.ExpandEnv(`

Aurora is a command line client for aurora daemon.

Usage:
  aurora [options] get [<package>]
  aurora [options] add <package>
  aurora [options] rm <package>
  aurora [options] log <package>
  aurora [options] watch <package> [-w]
  aurora [options] whoami
  aurora -h | --help
  aurora --version

Options:
  get                            Query specified package or query a list of packages.
  add                            Add a package to the queue.
  remove                         Remove a package from the queue.
  log                            Retrieve logs of a package.
  watch                          Watch build process.
  whoami                         Retrieves information about current using in the aurora.
  -a --address <rpc>             Address of aurorad rpc server. [default: https://aurora.reconquest.io/rpc/]
  -k --key <path>                Path to private RSA key. [default: $HOME/.config/aurora/id_rsa]
  --i-use-insecure-address       By default, aurora doesn't allow to use http:// schema in address.
                                  Use this flag to override this behavior.
  -w --wait                      Wait for a resulting status.
  -h --help                      Show this screen.
  --version                      Show version.
`)
)

type (
	Options struct {
		Get           bool
		Add           bool
		Rm            bool
		Log           bool
		Watch         bool
		Whoami        bool
		Address       string
		Package       string
		Key           string
		AllowInsecure bool `docopt:"--i-use-insecure-address"`
		Wait          bool
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

	err = validateAddress(opts)
	if err != nil {
		log.Fatalln(karma.Format(
			err,
			"invalid address (-a / --address) specified: %s", opts.Address,
		))
	}

	switch {
	case opts.Get:
		err = handleGet(opts)
	case opts.Add:
		err = handleAdd(opts)
	case opts.Rm:
		err = handleRemove(opts)
	case opts.Log:
		err = handleLog(opts)
	case opts.Watch:
		err = handleWatch(opts)
	case opts.Whoami:
		err = handleWhoami(opts)
	}

	if err != nil {
		log.Fatalln(err)
	}
}

func validateAddress(opts Options) error {
	uri, err := url.Parse(opts.Address)
	if err != nil {
		return karma.Format(
			err,
			"unable to parse URL",
		)
	}

	if uri.Scheme == "" {
		return fmt.Errorf("URL scheme is missing, add https:// to your address")
	}

	if uri.Scheme == "http" {
		if !opts.AllowInsecure {
			return fmt.Errorf(
				"insecure URL scheme specified, use https:// instead of http:// " +
					"or specify --i-use-insecure-address flag",
			)
		}
	} else if uri.Scheme != "https" {
		return fmt.Errorf(
			"unexpected URL scheme specified: %q://, use https://",
			uri.Scheme,
		)
	}

	if uri.Path == "" {
		return fmt.Errorf("URL path is not specified")
	}

	return nil
}
