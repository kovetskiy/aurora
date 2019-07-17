package main

import (
	jsonrpc "github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/reconquest/karma-go"

	"github.com/globalsign/mgo"
)

func NewRPCServer(collection *mgo.Collection, config *Config) (*jsonrpc.Server, error) {
	server := jsonrpc.NewServer()
	server.RegisterCodec(json2.NewCodec(), "application/json")

	auth, err := rpc.NewAuthService(config.AuthorizedKeysDir)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to initialize AuthService",
		)
	}

	pkg := rpc.NewPackageService(
		collection,
		auth,
		config.LogsDir,
		config.Instance,
	)

	server.RegisterService(auth, "AuthService")
	server.RegisterService(pkg, "PackageService")

	return server, nil
}
