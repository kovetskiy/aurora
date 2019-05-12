package main

import (
	"net/http"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/reconquest/karma-go"
)

type PackageService struct {
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

func NewPackageService(collection *mgo.Collection, config *Config) *PackageService {
	return &PackageService{
		collection: collection,
		config:     config,
	}
}

func (service *PackageService) ListPackages(
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

func (service *PackageService) GetPackage(
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
