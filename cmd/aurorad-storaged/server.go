package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/storage"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
)

const (
	repositoryArchiveFilename = "aurora.db.tar"
)

type Server struct {
	config  *config.Storage
	dbLock  sync.Mutex
	bus     *bus.Connection
	workers sync.WaitGroup
	queue   struct {
		archives bus.Consumer
	}
}

func (server *Server) Serve() error {
	err := server.initBus()
	if err != nil {
		return karma.Format(
			err,
			"unable to init bus",
		)
	}

	err = server.removeLock()
	if err != nil {
		return karma.Format(
			err,
			"unable to remove db lock",
		)
	}

	server.workers.Add(1)

	go server.serveQueue()

	server.workers.Wait()

	return nil
}

func (server *Server) initBus() error {
	log.Infof(
		karma.Describe("address", server.config.Bus),
		"connecting to bus",
	)

	conn, err := bus.Dial(server.config.Bus)
	if err != nil {
		return karma.Format(err, "can't dial bus")
	}

	log.Infof(nil, "connected to bus, creating a channel")

	channel, err := conn.Channel()
	if err != nil {
		return karma.Format(
			err,
			"unable to create bus channel",
		)
	}

	log.Infof(nil, "declaring queue consumer")

	server.queue.archives, err = channel.GetExchangeConsumer(
		bus.QueueArchives,
		server.config.Instance,
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to declare queue consumer",
		)
	}

	log.Infof(nil, "queue consumer %q declared", bus.QueueArchives)

	return nil
}

func (server *Server) serveQueue() {
	defer func() {
		server.workers.Done()
	}()

	for {
		delivery, ok := server.queue.archives.Consume()
		if !ok {
			break
		}

		var archive storage.Archive
		err := delivery.Decode(archive)
		if err != nil {
			log.Errorf(
				err,
				"bug: unable to decode archive item: %#v",
				delivery.GetBody(),
			)

			break
		}

		err = server.pull(archive)
		if err != nil {
			log.Errorf(err, "unable to download archive: %#v", archive)
		}
	}
}

func (server *Server) pull(archive storage.Archive) error {
	var (
		url      = "https://" + archive.Instance + "/" + archive.Archive
		filename = archive.Instance + "_" + filepath.Base(archive.Archive)
		path     = filepath.Join(server.config.Directory, archive.Archive)
	)

	log.Infof(
		karma.Describe("url", url).Describe("filename", filename),
		"downloading archive %s from %s",
		archive.Archive,
		archive.Instance,
	)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	request.Header.Set("User-Agent", "aurorad-storaged/"+version)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return karma.Format(
			err,
			"unable to make a request: %s", url,
		)
	}

	defer response.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		return karma.Format(
			err,
			"unable to create resulting archive",
		)
	}

	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		return karma.Format(
			err,
			"unable to copy response body to file",
		)
	}

	log.Infof(nil, "adding file to repo db: %s", path)

	err = server.repoAdd(path)
	if err != nil {
		return karma.Format(
			err,
			"unable to repo-add the archive",
		)
	}

	err = storage.CleanupRepositoryDirectory(
		server.config.Directory,
		archive.Package,
		server.config.History,
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to cleanup repository directory",
		)
	}

	return nil
}

func (server *Server) repoAdd(path string) error {
	server.dbLock.Lock()
	defer server.dbLock.Unlock()

	cmd := exec.Command(
		"repo-add",
		filepath.Join(server.config.Directory, repositoryArchiveFilename),
		path,
	)

	// TODO
	// meh? how to do that without that log.log.log?
	// don't like that this code knows about logger components
	err := lexec.NewExec(lexec.Loggerf(log.Logger.Log.Tracef), cmd).Run()
	if err != nil {
		return err
	}

	return nil
}

func (server *Server) removeLock() error {
	path := filepath.Join(server.config.Directory, repositoryArchiveFilename+".lck")

	log.Infof(nil, "ensuring database lock file does not exist: %s", path)

	raw, err := ioutil.ReadFile(path)
	if err != nil {
		// That's best case that lck file is not held by something
		if os.IsNotExist(err) {
			log.Infof(nil, "database lock file does not exist, proceeding")
			return nil
		}

		return karma.Format(
			err,
			"unable to open %s", path,
		)
	}

	log.Warningf(nil, "database lock file exists: %s", path)

	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		return karma.
			Describe("path", path).
			Format(
				err,
				"unexpected content in lck file: %q", string(raw),
			)
	}

	log.Warningf(nil, "database lock pid: %d", pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		return karma.Format(
			err,
			"unable to find process %d", pid,
		)
	}

	defer process.Release()

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		// process found, we can't remove lock
		return fmt.Errorf("process %d that locked %s is still running", pid, path)
	}

	log.Warningf(nil, "database lock process is not running: %d", pid)

	err = os.Remove(path)
	if err != nil {
		return karma.Format(
			err,
			"unable to remove lck file: %s",
			path,
		)
	}

	log.Warningf(nil, "database lock file has been removed: %s", path)

	return nil
}
