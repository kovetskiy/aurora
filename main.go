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
	"github.com/reconquest/faces"
	"github.com/reconquest/ser-go"
	"github.com/reconquest/threadpool-go"
)

var (
	version = "[manual build]"
	usage   = "aurora " + version + `

Usage:
  aurora [options] -L <address>
  aurora [options] -A <package>
  aurora [options] -R <package>
  aurora [options] -Q
  aurora [options] -P
  aurora -h | --help
  aurora --version

Options:
  -L --listen <address>   Listen specified address [default: :80].
  -A --add <package>      Add specified package to watch and make cycle queue.
  -R --remove <package>   Remove specified package from watch and make cycle queue.
  -P --process            Process watch and make cycle queue.
  -Q --query              Query package database.
  -i --interface <link>   Network host interface that shall be used in containers system.
                           [default: eth0]
  -t --threads <count>    Maximum amount of threads that can be used.
                           [default: 4]
  -d --database <path>    Path to place internal database file.
                           [default: /var/lib/aurora/aurora.db].
  -c --containers <path>  Root directory to place containers cloud.
                           [default: /var/lib/aurora/cloud/]
  -f --files <path>       Root directory that will be entirely copied into containers.
                           [default: /etc/aurora/container/]
  -s --filesystem <fs>    Pass specified option as hastur's filesystem.
                           [default: autodetect].
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
	var err error

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
	faces.SetLogger(logger)

	cloud, err = faces.NewHastur()
	if err != nil {
		fatalln(err)
	}
}

func main() {
	args := godocs.MustParse(usage, version, godocs.UsePager)

	bootstrap(args)

	database, err := openDatabase(args["--database"].(string))
	if err != nil {
		fatalh(err, "can't open aurora database")
	}

	switch {
	case args["--add"] != nil:
		err = addPackage(database, args["--add"].(string))

	case args["--remove"] != nil:
		err = removePackage(database, args["--remove"].(string))

	case args["--process"].(bool):
		err = processQueue(database, args)

	case args["--query"].(bool):
		err = queryPackage(database)

	case args["--listen"] != nil:
		err = serveWeb(
			database,
			args["--listen"].(string),
			args["--repository"].(string),
		)
	}

	if err != nil {
		fatalln(err)
	}
}

func addPackage(db *database, name string) error {
	db.set(
		name,
		pkg{Name: name, Status: "unknown", Date: time.Now()},
	)

	infof("package %s has been added", name)

	debugf("saving database")

	err := saveDatabase(db)
	if err != nil {
		return ser.Errorf(
			err, "can't save database",
		)
	}

	return nil
}

func removePackage(db *database, name string) error {
	db.remove(name)

	infof("package %s has been removed", name)

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
			pkg.Name, pkg.Version, pkg.Status, pkg.Date,
		)
	}

	return table.Flush()
}

func processQueue(db *database, args map[string]interface{}) error {
	var (
		cloudRoot       = args["--containers"].(string)
		cloudFileSystem = args["--filesystem"].(string)
		cloudNetwork    = args["--interface"].(string)
		containersDir   = args["--files"].(string)
		repositoryDir   = args["--repository"].(string)
		capacity        = argInt(args, "--threads")
	)

	pool := threadpool.New()
	pool.Spawn(capacity)

	infof(
		"thread pool with %d threads has been spawned", capacity,
	)

	cloud.SetHostNetwork(cloudNetwork)
	cloud.SetRootDirectory(cloudRoot)
	cloud.SetFileSystem(cloudFileSystem)
	cloud.SetQuietMode(true)

	err := os.MkdirAll(repositoryDir, 0644)
	if err != nil {
		return ser.Errorf(
			err, "can't mkdir %s", repositoryDir,
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
					sourcesDir:    containersDir,
					repositoryDir: repositoryDir,
					logger: logger.NewChildWithPrefix(
						fmt.Sprintf("(%s)", name),
					),
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
