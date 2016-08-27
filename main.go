package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
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
    aurora [options]
    aurora -h | --help
    aurora --version

Options:
  -L --listen <address>  Listen specified address [default: :80].
  -A --add <package>     Add specified package to watch and compile cycle queue.
  -R --remove <package>  Remove specified package from watch and compile cycle queue.
  -P --process           Process watch and compile cycle queue.
  -i --interface <link>  Specify host network interface for using in containers.
                          [default: eth0]
  -t --threads <n>       Specify amount of threads that can be used. [default: 4]
  -r --root <path>       Specify path to aurora root directory [default: /var/lib/aurora/].
  -f --files <path>      Specify container configuration files root directory
                          [default: /etc/aurora/conf.d/]
  -s --filesystem <fs>   Specify filesystem to use [default: autodetect].
  --debug                Show debug messages.
  --trace                Show trace messages.
  -h --help              Show this screen.
  --version              Show version.
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

	comm := exec.Command("x")
	comm.Run()
	bootstrap(args)

	database, err := openDatabase(
		filepath.Join(args["--root"].(string), "packages.db"),
	)
	if err != nil {
		fatalh(err, "can't open aurora database")
	}

	switch {
	case args["--add"] != nil:
		err = addPackage(args["--add"].(string), database)

	case args["--remove"] != nil:
		err = removePackage(args["--remove"].(string), database)

	case args["--process"].(bool):
		err = processQueue(
			database,
			args["--root"].(string),
			args["--files"].(string),
			args["--interface"].(string),
			args["--filesystem"].(string),
			argInt(args, "--threads"),
		)

	case args["--listen"] != nil:
		err = serveWeb(args["--listen"].(string), database)
	}

	if err != nil {
		fatalln(err)
	}
}

func addPackage(name string, db *database) error {
	db.add(name)

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

func removePackage(name string, db *database) error {
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

func processQueue(
	db *database,
	root string, files string,
	hostInterface string,
	filesystem string,
	capacity int,
) error {
	pool := threadpool.New()
	pool.Spawn(capacity)

	infof(
		"thread pool with %d threads has been spawned", capacity,
	)

	cloud.SetHostNetwork(hostInterface)
	cloud.SetRootDirectory(filepath.Join(root, "cloud"))
	cloud.SetQuietMode(true)
	cloud.SetFileSystem(filesystem)

	for {
		err := db.sync()
		if err != nil {
			return ser.Errorf(
				err, "can't sync database",
			)
		}

		debugf("database has been synchronized")

		for name, pkg := range db.getData() {
			tracef("pushing %s to thread pool queue", name)

			pool.Push(
				&build{
					database: db,
					pkg:      pkg,
					root:     root,
					files:    files,
					logger: logger.NewChildWithPrefix(
						fmt.Sprintf("(%s)", name),
					),
				},
			)
		}

		time.Sleep(time.Second * 5)
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
