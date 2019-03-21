package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aur-go"
	"github.com/kovetskiy/godocs"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/colorgful"
	karma "github.com/reconquest/karma-go"
	"github.com/reconquest/ser-go"
	"github.com/reconquest/threadpool-go"
)

var (
	version = "[manual build]"
	usage   = "aurora " + version + `

Usage:
  aurora [options] -L <address>
  aurora [options] -A <package>...
  aurora [options] -R <package>...
  aurora [options] -Q
  aurora [options] -P
  aurora -h | --help
  aurora --version

Options:
  -L --listen             Listen specified address [default: :80].
  -A --add                Add specified package to watch and make cycle queue.
  -R --remove             Remove specified package from watch and make cycle queue.
  -P --process            Process watch and make cycle queue.
  -Q --query              Query package database.
  -t --threads <count>    Maximum amount of threads that can be used.
                           [default: 0]
  -l --logs <path>        Root directory to place build logs.
                           [default: /var/log/aurora/packages/]
  -r --repository <path>  Root directory to place aurora repository.
                           [default: /srv/http/aurora/]
  --debug                 Show debug messages.
  --trace                 Show trace messages.
  -h --help               Show this screen.
  --version               Show version.
`
)

var logger = lorg.NewLog()

func bootstrap(args map[string]interface{}) {
	debugMode := args["--debug"].(bool)
	traceMode := args["--trace"].(bool)

	logger.SetFormat(
		colorgful.MustApplyDefaultTheme(
			"${time} ${level:[%s]:right:short} ${prefix}%s",
			colorgful.Dark,
		),
	)

	logger.SetIndentLines(true)

	if debugMode {
		logger.SetLevel(lorg.LevelDebug)
		logger.Debugf("debug mode enabled")
	}

	if traceMode {
		logger.SetLevel(lorg.LevelTrace)
		logger.Tracef("trace mode enabled")

		debugMode = true
	}

	aur.SetLogger(logger)
}

func main() {
	args := godocs.MustParse(usage, version, godocs.UsePager)

	bootstrap(args)

	database, err := NewDatabase("mongodb://localhost/aurora")
	if err != nil {
		fatalh(err, "can't open aurora database")
	}

	packages := database.C("packages")
	err = packages.EnsureIndex(mgo.Index{
		Key:    []string{"name"},
		Unique: true,
	})
	if err != nil {
		fatalh(err, "can't ensure index for collection")
	}

	switch {
	case args["--add"].(bool):
		err = addPackage(packages, args["<package>"].([]string))

	case args["--remove"].(bool):
		err = removePackage(packages, args["<package>"].([]string))

	case args["--process"].(bool):
		err = processQueue(packages, args)

	case args["--query"].(bool):
		err = queryPackage(packages)

	case args["--listen"].(bool):
		err = serveWeb(
			packages,
			args["<address>"].(string),
			args["--repository"].(string),
			args["--logs"].(string),
		)
	}

	if err != nil {
		fatalln(err)
	}
}

func addPackage(collection *mgo.Collection, packages []string) error {
	var err error

	for _, name := range packages {
		err = collection.Insert(
			pkg{Name: name, Status: StatusUnknown, Date: time.Now()},
		)

		if err == nil {
			infof("package %s has been added", name)
		} else if mgo.IsDup(err) {
			warningf("package %s has not been added: already exists", name)
		} else {
			return err
		}
	}

	return nil
}

func removePackage(collection *mgo.Collection, packages []string) error {
	var err error

	for _, name := range packages {
		err = collection.Remove(
			bson.M{"name": name},
		)

		if err == nil {
			infof("package %s has been removed", name)
		} else if err == mgo.ErrNotFound {
			warningf("package %s not found", name)
		} else {
			return err
		}
	}

	return nil
}

func queryPackage(collection *mgo.Collection) error {
	var (
		pkg      = pkg{}
		packages = collection.Find(bson.M{}).Iter()
		table    = tabwriter.NewWriter(os.Stdout, 1, 4, 1, ' ', 0)
	)

	for packages.Next(&pkg) {
		fmt.Fprintf(
			table,
			"%s\t%s\t%s\t%s\n",
			pkg.Name, pkg.Version, pkg.Status,
			pkg.Date.Format("2006-01-02 15:04:05"),
		)
	}

	return table.Flush()
}

func processQueue(collection *mgo.Collection, args map[string]interface{}) error {
	var (
		repositoryDir = args["--repository"].(string)
		logsDir       = args["--logs"].(string)
		capacity      = argInt(args, "--threads")
	)

	if capacity == 0 {
		capacity = runtime.NumCPU()
	}

	pool := threadpool.New()
	pool.Spawn(capacity)

	infof(
		"thread pool with %d threads has been spawned", capacity,
	)

	err := os.MkdirAll(repositoryDir, 0644)
	if err != nil {
		return ser.Errorf(
			err, "can't mkdir %s", repositoryDir,
		)
	}

	err = os.MkdirAll(logsDir, 0755)
	if err != nil {
		return karma.Format(
			err,
			"unable to mkdir logs directory: %s", logsDir,
		)
	}

	for {
		pkg := pkg{}
		packages := collection.Find(bson.M{}).Iter()

		for packages.Next(&pkg) {
			switch pkg.Status {

			case StatusProcessing:
				if time.Since(pkg.Date).Hours() < 1 {
					continue
				}

			case StatusSuccess:
				if time.Since(pkg.Date).Hours() < 4 {
					continue
				}

			case StatusFailure:
				if time.Since(pkg.Date).Hours() < 1 {
					continue
				}
			}

			tracef("pushing %s to thread pool queue", pkg.Name)

			pool.Push(
				&build{
					collection:    collection,
					pkg:           pkg,
					repositoryDir: repositoryDir,
					logsDir:       logsDir,
				},
			)
		}

		time.Sleep(time.Second * 2)
	}

	return nil
}

func argInt(args map[string]interface{}, arg string) int {
	value, err := strconv.Atoi(args[arg].(string))
	if err != nil {
		fatalh(
			err,
			"invalid value %q passed in %s option, should be an integer",
			args[arg], arg,
		)
		return 0
	}

	return value
}
