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

type RPCLogsService struct {
	collection *mgo.Collection
	config     *Config
}

type RequestGetLogs struct {
	Name string `json:"name"`
}

type ResponseGetLogs struct {
	Logs string `json:"logs"`
}

type RequestFollowLogs struct {
	Name string `json:"name"`
}

type ResponseFollowLogs struct {
	Stream string `json:"stream"`
}

func NewRPCLogsService(collection *mgo.Collection, config *Config) *RPCLogsService {
	return &RPCLogsService{
		collection: collection,
		config:     config,
	}
}

func (service *RPCLogsService) GetLogs(
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

func (service *RPCLogsService) FollowLogs(
	source *http.Request,
	request *RequestFollowLogs,
	response *ResponseFollowLogs,
) error {
	var pkg Package
	err := service.collection.Find(
		bson.M{"name": request.Name},
	).One(&pkg)
	if err == mgo.ErrNotFound {
		return nil
	}

	// here can be complex logic with retrieving address of processor,
	address := "ws://" + pkg.Instance + ":" + "9999" + "/?package=" + request.Name

	response.Stream = address

	return nil
}
