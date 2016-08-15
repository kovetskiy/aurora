package main

import (
	"strconv"
	"time"

	"github.com/kovetskiy/godocs"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/colorgful"
	"github.com/seletskiy/hierr"
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
  -Q --queue             Process watch and compile cycle queue.
  -t --threads <n>       Specify amount of threads that should be used.
  -d --database <path>   Specify path to aurora database [default: /var/lib/aurora/].
  -h --help              Show this screen.
  --version              Show version.
`
)

var (
	logger    = lorg.NewLog()
	debugMode = false
)

func main() {
	args := godocs.MustParse(usage, version, godocs.UsePager)

	logger.SetFormat(
		colorgful.MustApplyDefaultTheme(
			"${time} ${level:[%s]:right:short} ${prefix}%s",
			colorgful.Dark,
		),
	)

	debugMode = args["--debug"].(bool)
	if debugMode {
		logger.SetLevel(lorg.LevelDebug)
	}

	database, err := openDatabase(args["--database"].(string))
	if err != nil {
		fatalh(err, "can't open database at %s", args["--database"].(string))
	}

	threads, err := strconv.Atoi(args["--threads"].(string))
	if err != nil {
		fatalh(err, "invalid threads count passed in --threads")
	}

	switch {
	case args["--queue"].(bool):
		err = processQueue(database, threads)

	case args["--listen"] != nil:
		err = serveWeb(args["--listen"].(string), database)

	case args["--add"] != nil:
		err = addPackage(args["--add"].(string), database)

	case args["--remove"] != nil:
		err = removePackage(args["--remove"].(string), database)
	}

	if err != nil {
		fatalln(err)
	}
}

func addPackage(name string, db *database) error {
	panic("x")
}

func removePackage(name string, db *database) error {
	panic("x")
}

func processQueue(db *database, threads int) error {
	queue := newQueue(threads)

	for range time.Tick(time.Minute) {
		err := db.sync()
		if err != nil {
			return hierr.Errorf(
				err, "can't sync database",
			)
		}

		for _, pkg := range db.getData() {
			queue.push(&build{
				database: db,
				pkg:      pkg,
			})
		}
	}

	return nil
}
