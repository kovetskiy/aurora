package main

import (
	"github.com/gorilla/rpc"

	"github.com/globalsign/mgo"
	"github.com/gorilla/rpc/json"
	"github.com/kovetskiy/aurora/pkg/proto"
)

func NewRPCServer(collection *mgo.Collection, config *Config) *rpc.Server {
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")

	server.RegisterService(
		proto.NewPackageService(collection, config.LogsDir),
		"PackageService",
	)

	return server
}
