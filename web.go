package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"

	"github.com/globalsign/mgo"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const (
	staticPrefix = "/aurora"
)

type Web struct {
	collection *mgo.Collection
	config     *Config
	static     http.Handler
}

func serveWeb(collection *mgo.Collection, config *Config) error {
	web := &Web{
		collection: collection,
		config:     config,
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	web.initStatic()
	router.Get(staticPrefix+"/*", web.static.ServeHTTP)

	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")

	server.RegisterService(
		NewPackageService(collection, config),
		"PackageService",
	)

	server.RegisterService(
		NewLogsService(collection, config),
		"LogsService",
	)

	router.Post("/rpc/", server.ServeHTTP)
	router.Get("/rpc/", server.ServeHTTP)

	infof("listening at %s", config.Listen)

	return http.ListenAndServe(config.Listen, router)
}

func (web *Web) initStatic() {
	web.static = http.StripPrefix(
		staticPrefix,
		http.FileServer(http.Dir(web.config.RepoDir)),
	)
}

func (web *Web) Error(writer http.ResponseWriter, err error) {
	writer.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(writer, err.Error())
}
