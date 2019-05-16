package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/reconquest/karma-go"
)

var (
	DefaultBusServerPort = 4242
)

type RPCPackageService struct {
	collection *mgo.Collection
	config     *Config
}

type RequestListPackages struct {
	//
}

type ResponseListPackages struct {
	Packages []Package `json:"packages"`
}

type RequestGetPackage struct {
	Name string `json:"name"`
}

type ResponseGetPackage struct {
	Package *Package `json:"package"`
}

type RequestGetLogs struct {
	Name string `json:"name"`
}

type ResponseGetLogs struct {
	Logs string `json:"logs"`
}

type RequestGetBus struct {
	Name string `json:"name"`
}

type ResponseGetBus struct {
	Stream string `json:"stream"`
}

func NewRPCPackageService(
	collection *mgo.Collection,
	config *Config,
) *RPCPackageService {
	return &RPCPackageService{
		collection: collection,
		config:     config,
	}
}

func (service *RPCPackageService) ListPackages(
	source *http.Request,
	request *RequestListPackages,
	response *ResponseListPackages,
) error {
	err := service.collection.Find(bson.M{}).All(&response.Packages)
	if err != nil {
		return karma.Format(
			err,
			"unable to find packages in database",
		)
	}

	return nil
}

func (service *RPCPackageService) GetPackage(
	source *http.Request,
	request *RequestGetPackage,
	response *ResponseGetPackage,
) error {
	err := service.collection.Find(
		bson.M{"name": request.Name},
	).One(&response.Package)
	if err == mgo.ErrNotFound {
		response.Package = nil
		return nil
	}
	if err != nil {
		return karma.Format(
			err,
			"unable to find package in database",
		)
	}

	return nil
}

func (service *RPCPackageService) GetLogs(
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

func (service *RPCPackageService) GetBus(
	source *http.Request,
	request *RequestGetBus,
	response *ResponseGetBus,
) error {
	var pkg Package
	err := service.collection.Find(
		bson.M{"name": request.Name},
	).One(&pkg)
	if err == mgo.ErrNotFound {
		return nil
	}

	// here can be complex logic with retrieving address of processor
	address := fmt.Sprintf(
		"ws://%s:%d/?package=%s",
		pkg.Instance,
		DefaultBusServerPort,
		request.Name,
	)

	response.Stream = address

	return nil
}
