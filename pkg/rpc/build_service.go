package rpc

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/kovetskiy/aurora/pkg/proto"
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

	fmt.Fprintf(os.Stderr, "XXXXXX build_service.go:33 signer.Name: %#v\n", signer.Name)

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
		return err
	}

	return nil
}
