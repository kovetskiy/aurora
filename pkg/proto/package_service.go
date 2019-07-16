package proto

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/aurora"
	"github.com/reconquest/karma-go"
)

var (
	DefaultBusServerPort = 4242
)

type PackageService struct {
	collection *mgo.Collection
	logsDir    string
}

type RequestListPackages struct {
	//
}

type ResponseListPackages struct {
	Packages []aurora.Package `json:"packages"`
}

type RequestGetPackage struct {
	Name string `json:"name"`
}

type ResponseGetPackage struct {
	Package *aurora.Package `json:"package"`
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

func NewPackageService(
	collection *mgo.Collection,
	logsDir string,
) *PackageService {
	return &PackageService{
		collection: collection,
		logsDir:    logsDir,
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

func (service *PackageService) GetLogs(
	source *http.Request,
	request *RequestGetLogs,
	response *ResponseGetLogs,
) error {
	var pkg aurora.Package
	err := service.collection.Find(
		bson.M{"name": request.Name},
	).One(&pkg)
	if err == mgo.ErrNotFound {
		return nil
	}

	contents, err := ioutil.ReadFile(
		filepath.Join(service.logsDir, pkg.Name),
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

func (service *PackageService) GetBus(
	source *http.Request,
	request *RequestGetBus,
	response *ResponseGetBus,
) error {
	var pkg aurora.Package
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
