package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kovetskiy/ko"
	"github.com/reconquest/ser-go"
)

type pkg struct {
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Status  string    `json:"status"`
	Date    time.Time `json:"date"`
}

type database struct {
	*sync.RWMutex
	path string
	data map[string]pkg
}

const (
	StatusUnknown    = "unknown"
	StatusFailure    = "failure"
	StatusSuccess    = "success"
	StatusProcessing = "processing"
)

func openDatabase(path string) (*database, error) {
	database := &database{
		RWMutex: &sync.RWMutex{},
		path:    path,
		data:    map[string]pkg{},
	}

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(path), 0644)
		if err != nil {
			return nil, ser.Errorf(
				err, "can't create directory for aurora database",
			)
		}

		err = ioutil.WriteFile(path, []byte(`{}`), 0600)
		if err != nil {
			return nil, ser.Errorf(
				err, "can't initialize empty database file",
			)
		}

		return database, nil
	}

	err = ko.Load(path, &database.data, json.Unmarshal)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func (database *database) getData() map[string]pkg {
	database.RLock()
	defer database.RUnlock()

	new := map[string]pkg{}
	for key, value := range database.data {
		new[key] = value
	}
	return new
}

func (database *database) sync() error {
	database.Lock()
	defer database.Unlock()

	err := ko.Load(database.path, &database.data, json.Unmarshal)
	if err != nil {
		return err
	}

	return nil
}

func (database *database) set(name string, pkg pkg) {
	database.Lock()
	defer database.Unlock()

	database.data[name] = pkg
}

func (database *database) remove(name string) {
	database.Lock()
	defer database.Unlock()

	delete(database.data, name)
}

func (database *database) get(name string) (pkg, bool) {
	database.RLock()
	defer database.RUnlock()

	pkg, ok := database.data[name]

	return pkg, ok
}

func saveDatabase(database *database) error {
	database.RLock()
	defer database.RUnlock()

	output, err := json.MarshalIndent(database.data, "", "    ")
	if err != nil {
		return ser.Errorf(
			err, "can't marshal database data",
		)
	}

	err = ioutil.WriteFile(database.path, output, 0600)
	if err != nil {
		return ser.Errorf(
			err, "can't write database file",
		)
	}

	return nil
}
