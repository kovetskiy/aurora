package main

import (
	"net/http"

	"github.com/globalsign/mgo"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const (
	staticPrefix = "/aurora"
)

type Web struct {
	static http.Handler
}

func serveWeb(collection *mgo.Collection, config *Config) error {
	web := &Web{}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	web.initStatic(config)

	router.Get(staticPrefix+"/*", web.static.ServeHTTP)

	rpc := NewRPCServer(collection, config)
	router.Post("/rpc/", rpc.ServeHTTP)

	infof("listening at %s", config.Listen)

	return http.ListenAndServe(config.Listen, router)
}

func (web *Web) initStatic(config *Config) {
	web.static = http.StripPrefix(
		staticPrefix,
		http.FileServer(http.Dir(config.RepoDir)),
	)
}
