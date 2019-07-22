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

var (
	ErrorUnauthorized = errors.New(
		"you are not authorized to perform this action",
	)
)

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
	pkgs     *mgo.Collection
	auth     *AuthService
	logsDir  string
	instance string
}

func NewPackageService(
	pkgs *mgo.Collection,
	auth *AuthService,
	logsDir string,
	instance string,
) *PackageService {
	return &PackageService{
		pkgs:     pkgs,
		logsDir:  logsDir,
		auth:     auth,
		instance: instance,
	}
}

func (service *PackageService) ListPackages(
	source *http.Request,
	request *proto.RequestListPackages,
	response *proto.ResponseListPackages,
) error {
	err := service.pkgs.Find(bson.M{}).All(&response.Packages)
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
	err := service.pkgs.Find(
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
	var pkg proto.Package
	err := service.pkgs.Find(
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
	var pkg proto.Package
	err := service.pkgs.Find(
		bson.M{"name": request.Name},
	).One(&pkg)
	if err == mgo.ErrNotFound {
		return errors.New("no such package")
	}

	instance := service.instance

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
	if !proto.IsValidPackageName(request.Name) {
		return errors.New("invalid package name")
	}

	err := service.pkgs.Insert(
		proto.Package{
			Name:      request.Name,
			Status:    proto.PackageStatusQueued,
			UpdatedAt: time.Now(),
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

	err := service.pkgs.Remove(
		bson.M{"name": request.Name},
	)

	return err
}
