package database

import (
	"time"

	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/reconquest/karma-go"

	"github.com/globalsign/mgo"
)

type Database struct {
	*mgo.Database

	dsn     string
	session *mgo.Session
}

func NewDatabase(dsn string) (*Database, error) {
	db := &Database{dsn: dsn}

	err := db.Connect()
	if err != nil {
		return nil, err
	}

	go db.Watch()

	return db, nil
}

func (db *Database) Connect() error {
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

func (db *Database) Watch() {
	for {
		time.Sleep(time.Second * 1)

		err := db.session.Ping()
		if err != nil {
			log.Errorf(err, "unable to ping db")
		} else {
			continue
		}

		log.Warning("db connection has gone away, trying to reconnect")

		err = db.Connect()
		if err != nil {
			log.Errorf(err, "can't establish db connection")
			continue
		}

		log.Info("db connection has been re-established")
	}
}

func (db *Database) Packages() *mgo.Collection {
	return db.C("packages")
}
