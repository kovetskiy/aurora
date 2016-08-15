package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/kovetskiy/ko"
)

type pkg struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	LastBuild time.Time `json:"last_build"`
}

type database struct {
	*sync.RWMutex
	path string
	data map[string]pkg
}

func openDatabase(path string) (*database, error) {
	database := &database{
		RWMutex: &sync.RWMutex{},
		path:    path,
		data:    map[string]pkg{},
	}

	err := ko.Load(path, &database.data, json.Unmarshal)
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
