package main

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

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
                           [default: 4]
  -d --database <path>    Path to place internal database file.
                           [default: /var/lib/aurora/aurora.db].
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

	database, err := openDatabase(args["--database"].(string))
	if err != nil {
		fatalh(err, "can't open aurora database")
	}

	switch {
	case args["--add"].(bool):
		err = addPackage(database, args["<package>"].([]string))

	case args["--remove"].(bool):
		err = removePackage(database, args["<package>"].([]string))

	case args["--process"].(bool):
		err = processQueue(database, args)

	case args["--query"].(bool):
		err = queryPackage(database)

	case args["--listen"].(bool):
		err = serveWeb(
			database,
			args["<address>"].(string),
			args["--repository"].(string),
			args["--logs"].(string),
		)
	}

	if err != nil {
		fatalln(err)
	}
}

func addPackage(db *database, packages []string) error {
	for _, name := range packages {
		db.set(
			name,
			pkg{Name: name, Status: "unknown", Date: time.Now()},
		)

		infof("package %s has been added", name)
	}

	debugf("saving database")

	err := saveDatabase(db)
	if err != nil {
		return ser.Errorf(
			err, "can't save database",
		)
	}

	return nil
}

func removePackage(db *database, packages []string) error {
	for _, name := range packages {
		db.remove(name)

		infof("package %s has been removed", name)
	}

	debugf("saving database")

	err := saveDatabase(db)
	if err != nil {
		return ser.Errorf(
			err, "can't save database",
		)
	}

	return nil
}

func queryPackage(db *database) error {
	table := tabwriter.NewWriter(os.Stdout, 1, 4, 1, ' ', 0)
	for _, pkg := range db.getData() {
		fmt.Fprintf(
			table,
			"%s\t%s\t%s\t%s\n",
			pkg.Name, pkg.Version, pkg.Status,
			pkg.Date.Format("2006-01-02 15:04:05"),
		)
	}

	return table.Flush()
}

func processQueue(db *database, args map[string]interface{}) error {
	var (
		repositoryDir = args["--repository"].(string)
		logsDir       = args["--logs"].(string)
		capacity      = argInt(args, "--threads")
	)

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
		err := db.sync()
		if err != nil {
			return ser.Errorf(
				err, "can't synchronize database",
			)
		}

		debugf("database has been synchronized")

		for name, pkg := range db.getData() {
			tracef("pushing %s to thread pool queue", name)

			pool.Push(
				&build{
					database:      db,
					pkg:           pkg,
					repositoryDir: repositoryDir,
					logsDir:       logsDir,
				},
			)
		}

		time.Sleep(time.Minute * 10)
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
