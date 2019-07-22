package main

import (
	"net/http"

	"github.com/globalsign/mgo"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	jsonrpc "github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"
	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/reconquest/karma-go"
)

func listenAndServe(
	pkgs *mgo.Collection,
	builds *mgo.Collection,
	config *config.RPC,
) error {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	rpc, err := newRPCServer(pkgs, builds, config)
	if err != nil {
		return karma.Format(
			err,
			"unable to create RPC server",
		)
	}

	router.Post("/rpc/", rpc.ServeHTTP)

	log.Infof(nil, "listening at %s", config.Listen)

	return http.ListenAndServe(config.Listen, router)
}

func newRPCServer(
	pkgs, builds *mgo.Collection,
	config *config.RPC,
) (*jsonrpc.Server, error) {
	server := jsonrpc.NewServer()
	server.RegisterCodec(json2.NewCodec(), "application/json")

	authService, err := rpc.NewAuthService(config.AuthorizedKeysDir)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to initialize AuthService",
		)
	}

	packageService := rpc.NewPackageService(
		pkgs,
		authService,
		"",
		"",
	)

	buildService := rpc.NewBuildService(
		builds,
		authService,
	)

	server.RegisterService(authService, "AuthService")
	server.RegisterService(packageService, "PackageService")
	server.RegisterService(buildService, "BuildService")

	return server, nil
}
