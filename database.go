package main

import (
	"time"

	"github.com/globalsign/mgo"
	karma "github.com/reconquest/karma-go"
	"gitlab.com/reconquest/lablab/log"
)

type Database struct {
	*mgo.Database

	dsn     string
	session *mgo.Session
}

type pkg struct {
	Name    string    `bson:"name"`
	Version string    `bson:"version"`
	Status  string    `bson:"status"`
	Date    time.Time `bson:"date"`
}

const (
	StatusUnknown    = "unknown"
	StatusFailure    = "failure"
	StatusSuccess    = "success"
	StatusProcessing = "processing"
)

func NewDatabase(dsn string) (*Database, error) {
	db := &Database{dsn: dsn}

	err := db.connect()
	if err != nil {
		return nil, err
	}

	go db.watch()

	return db, nil
}

func (db *Database) connect() error {
	log.Infof(
		nil,
		"connecting to db %q",
		db.dsn,
	)

	started := time.Now()

	session, err := mgo.Dial(db.dsn)
	if err != nil {
		return karma.Format(
			err,
			"unable to connect to db: %s",
			db.dsn,
		)
	}

	log.Infof(nil, "db connected | took %s", time.Since(started))

	db.session = session

	db.Database = session.DB("")

	return nil
}

func (db *Database) watch() {
	for {
		time.Sleep(time.Second * 1)

		err := db.session.Ping()
		if err != nil {
			log.Error(karma.Format(err, "unable to ping db"))
		} else {
			continue
		}

		log.Warning("db connection has gone away, trying to reconnect")

		err = db.connect()
		if err != nil {
			log.Error(karma.Format(err, "can't establish db connection"))
			continue
		}

		log.Info("db connection has been re-established")
	}
}
