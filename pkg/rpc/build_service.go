package rpc

import (
	"errors"
	"net/http"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/reconquest/karma-go"
)

type BuildService struct {
	auth   *AuthService
	builds *mgo.Collection
}

func NewBuildService(builds *mgo.Collection, auth *AuthService) *BuildService {
	return &BuildService{
		builds: builds,
		auth:   auth,
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

	_, err := service.builds.Upsert(bson.M{
		"instance": build.Instance,
		"package":  build.Package,
	}, build)
	if err != nil {
		return karma.Format(
			err,
			"unable to upsert a record in database",
		)
	}

	return nil
}
