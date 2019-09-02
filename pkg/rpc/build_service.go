package rpc

import (
	"errors"
	"net/http"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/storage"
	"github.com/reconquest/karma-go"
)

type BuildService struct {
	auth     *AuthService
	builds   *mgo.Collection
	archives bus.Publisher
}

func NewBuildService(
	builds *mgo.Collection,
	auth *AuthService,
	archives bus.Publisher,
) *BuildService {
	return &BuildService{
		builds:   builds,
		auth:     auth,
		archives: archives,
	}
}

func (service *BuildService) PushBuild(
	source *http.Request,
	request *proto.RequestPushBuild,
	response *proto.ResponsePushBuild,
) error {
	signer := service.auth.Verify(request.Signature)
	if signer == nil {
		return ErrorUnauthorized
	}

	build := request.Build
	if !proto.IsValidPackageName(build.Package) {
		return errors.New("invalid package name")
	}

	build.Instance = signer.Name

	err := build.Validate()
	if err != nil {
		return err
	}

	log.Debugf(build.Describe(), "upserting build")

	_, err = service.builds.Upsert(bson.M{
		"instance": build.Instance,
		"package":  build.Package,
	}, build)
	if err != nil {
		return karma.Format(
			err,
			"unable to upsert a record in database",
		)
	}

	if build.Status == proto.PackageStatusSuccess {
		log.Infof(build.Describe(), "publishing archive")
		err = service.archives.Publish(storage.Archive{
			Instance: build.Instance,
			Package:  build.Package,
			Archive:  build.Archive,
		})
		if err != nil {
			return karma.Format(
				err,
				"unable to publish archive to the queue",
			)
		}
	}

	return nil
}
