package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
	"github.com/reconquest/regexputil-go"
)

const (
	Lifetime = time.Hour * 96
)

func (proc *Processor) Cleanup() error {
	globbed, err := filepath.Glob(
		filepath.Join(proc.repoDir, "*.pkg.*"),
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to glob for packages",
		)
	}

	type archive struct {
		Path     string
		Name     string
		Time     string
		Basename string
	}

	var removed int64
	dbstate := map[string]bool{}
	lockMutex := &sync.Mutex{}
	for _, fullpath := range globbed {
		basename := filepath.Base(fullpath)

		matches := reArchiveFilename.FindStringSubmatch(basename)

		name := regexputil.Subexp(reArchiveFilename, matches, "name")
		built := regexputil.Subexp(reArchiveFilename, matches, "time")

		unixBuilt, err := strconv.Atoi(built)
		if err != nil {
			infof("cleanup: broken name | %s | %s", name, fullpath)

			err := proc.removeArchive(lockMutex, fullpath)
			if err != nil {
				logger.Error(err)
			}

			continue
		}

		builtAt := time.Unix(int64(unixBuilt), 0)

		if time.Now().Sub(builtAt) > Lifetime {
			infof("cleanup: too old | %s | %s", name, fullpath)

			err := proc.removeArchive(lockMutex, fullpath)
			if err != nil {
				logger.Error(err)
			} else {
				removed++
			}

			continue
		}

		present, ok := dbstate[name]
		if !ok {
			count, err := proc.storage.Find(bson.M{"name": name}).Count()
			if err != nil {
				logger.Fatal(err)
				continue
			}

			dbstate[name] = count > 0
		}

		if !present {
			infof("cleanup: not-present | %s | %s", name, fullpath)

			err = proc.removeArchive(lockMutex, fullpath)
			if err != nil {
				logger.Error(err)
				continue
			}

			removed++
		}
	}

	infof("cleanup: removed %d archives", removed)

	return nil
}

func (proc *Processor) removeArchive(lock *sync.Mutex, path string) error {
	lock.Lock()

	cmd := exec.Command("repo-remove", filepath.Join(proc.repoDir, packagesDatabaseFile), path)
	err := lexec.NewExec(lexec.Loggerf(logger.Tracef), cmd).Run()

	lock.Unlock()

	if err != nil {
		if !strings.Contains(err.Error(), "No packages modified, nothing to do") {
			return err
		}
	}

	err = os.Remove(path)
	if err != nil {
		return karma.Format(err, "unable to rm: %s", path)
	}

	return nil
}
