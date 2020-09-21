package rpc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/reconquest/karma-go"
)

var ErrorUnauthorized = errors.New("you are not authorized to perform this action")

// PackageService handles all interactions with packages, including:
//
// - adding/removing a package to the queue
// - retrieving list of packages
// - retrieving info about a package
// - retrieving logs after build
// - watching logs from bus
//
// Should be splitted into several services in order to decrease
// responsibilities.
type PackageService struct {
	collection *mgo.Collection
	auth       *AuthService
	logsDir    string
	instance   string
}

func NewPackageService(
	collection *mgo.Collection,
	auth *AuthService,
	logsDir string,
	instance string,
) *PackageService {
	return &PackageService{
		collection: collection,
		logsDir:    logsDir,
		auth:       auth,
		instance:   instance,
	}
}

func (service *PackageService) ListPackages(
	source *http.Request,
	request *proto.RequestListPackages,
	response *proto.ResponseListPackages,
) error {
	signer := service.auth.Verify(request.Signature)
	if signer == nil {
		return ErrorUnauthorized
	}

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
	request *proto.RequestGetPackage,
	response *proto.ResponseGetPackage,
) error {
	signer := service.auth.Verify(request.Signature)
	if signer == nil {
		return ErrorUnauthorized
	}

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
	request *proto.RequestGetLogs,
	response *proto.ResponseGetLogs,
) error {
	signer := service.auth.Verify(request.Signature)
	if signer == nil {
		return ErrorUnauthorized
	}

	var pkg proto.Package
	err := service.collection.Find(
		bson.M{"name": request.Name},
	).One(&pkg)
	if err == mgo.ErrNotFound {
		return errors.New("no such package")
	}

	if !proto.IsValidPackageName(pkg.Name) {
		return errors.New("invalid package name in database found")
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
	request *proto.RequestGetBus,
	response *proto.ResponseGetBus,
) error {
	signer := service.auth.Verify(request.Signature)
	if signer == nil {
		return ErrorUnauthorized
	}

	var pkg proto.Package
	err := service.collection.Find(
		bson.M{"name": request.Name},
	).One(&pkg)
	if err == mgo.ErrNotFound {
		return errors.New("no such package")
	}

	instance := pkg.Instance
	if instance == "" {
		instance = service.instance
	}

	// here can be complex logic with retrieving address of processor
	address := fmt.Sprintf(
		"ws://%s:%d/?package=%s",
		instance,
		proto.DefaultBusServerPort,
		request.Name,
	)

	response.Stream = address

	return nil
}

func (service *PackageService) AddPackage(
	source *http.Request,
	request *proto.RequestAddPackage,
	response *proto.ResponseAddPackage,
) error {
	signer := service.auth.Verify(request.Signature)
	if signer == nil {
		return ErrorUnauthorized
	}

	if !proto.IsValidPackageName(request.Name) {
		return errors.New("invalid package name")
	}

	err := service.collection.Insert(
		proto.Package{
			Name:     request.Name,
			Status:   proto.BuildStatusQueued.String(),
			Date:     time.Now(),
			CloneURL: request.CloneURL,
			Subdir:   request.Subdir,
		},
	)

	if err == nil {
		return nil
	} else if mgo.IsDup(err) {
		return nil
	} else {
		return err
	}
}

func (service *PackageService) RemovePackage(
	source *http.Request,
	request *proto.RequestRemovePackage,
	response *proto.ResponseRemovePackage,
) error {
	signer := service.auth.Verify(request.Signature)
	if signer == nil {
		return ErrorUnauthorized
	}

	err := service.collection.Remove(
		bson.M{"name": request.Name},
	)

	return err
}
