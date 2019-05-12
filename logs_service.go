package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/reconquest/karma-go"
)

type LogsService struct {
	collection *mgo.Collection
	config     *Config
}

type RequestGetLogs struct {
	Name string `json:"name"`
}

type ResponseGetLogs struct {
	Logs string `json:"logs"`
}

func NewLogsService(collection *mgo.Collection, config *Config) *LogsService {
	return &LogsService{
		collection: collection,
		config:     config,
	}
}

func (service *LogsService) GetLogs(
	source *http.Request,
	request *RequestGetLogs,
	response *ResponseGetLogs,
) error {
	var pkg Package
	err := service.collection.Find(
		bson.M{"name": request.Name},
	).One(&pkg)
	if err == mgo.ErrNotFound {
		return nil
	}

	contents, err := ioutil.ReadFile(
		filepath.Join(service.config.LogsDir, pkg.Name),
	)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return karma.Format(
			err,
			"unable to read logs file",
		)
	}

	response.Logs = string(contents)

	return nil
}
