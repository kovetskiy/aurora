package main

import (
	"net/http"

	"github.com/globalsign/mgo"
	"github.com/reconquest/karma-go"
)

func processQueue(storage *mgo.Collection, config *Config) error {
	bus := NewBus()

	processor := NewProcessor(storage, config, bus)
	busServer := NewBusServer(bus)

	err := processor.Init()
	if err != nil {
		return karma.Format(
			err,
			"unable to initialize queue processor",
		)
	}

	go processor.Process()

	infof("starting bus server at %s", config.Bus.Listen)

	err = http.ListenAndServe(config.Bus.Listen, busServer)
	if err != nil {
		return karma.Format(
			err,
			"unable to listen and serve bus server at %s",
			config.Bus.Listen,
		)
	}

	return nil
}
