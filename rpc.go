package main

import (
	"github.com/globalsign/mgo"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

func NewRPCServer(collection *mgo.Collection, config *Config) *rpc.Server {
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")

	server.RegisterService(
		NewRPCPackageService(collection, config),
		"PackageService",
	)

	server.RegisterService(
		NewRPCLogsService(collection, config),
		"LogsService",
	)

	return server
}
