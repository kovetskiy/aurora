package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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

	router.Route("/api/v1/", func(api chi.Router) {
		api.Get("/pkg/", web.handlePackagesList)

		api.Route("/pkg/{name}", func(single chi.Router) {
			single.Use(web.withPackage)

			single.Get("/status", web.handlePackageStatus)
			single.Get("/logs", web.handlePackageLogs)
			single.Post("/build", web.handlePackageQueue)
		})
	})

	infof("listening at %s", config.Listen)

	return http.ListenAndServe(config.Listen, router)
}

func (web *Web) initStatic() {
	web.static = http.StripPrefix(
		staticPrefix,
		http.FileServer(http.Dir(web.config.RepoDir)),
	)
}

func (web *Web) withPackage(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			pkg := &pkg{}

			err := web.collection.Find(
				bson.M{"name": chi.URLParam(request, "name")},
			).One(&pkg)
			if err == mgo.ErrNotFound {
				http.Error(writer, "no such package", http.StatusNotFound)
				return
			}

			nextContext := context.WithValue(request.Context(), "pkg", pkg)

			next.ServeHTTP(writer, request.WithContext(nextContext))
		},
	)
}

func (web *Web) handlePackagesList(
	writer http.ResponseWriter,
	request *http.Request,
) {
	packages := []pkg{}
	err := web.collection.Find(bson.M{}).All(&packages)
	if err != nil {
		web.Error(writer, err)
		return
	}

	for _, pkg := range packages {
		fmt.Fprintln(writer, pkg.Name)
	}
}

func (web *Web) handlePackageStatus(
	writer http.ResponseWriter,
	request *http.Request,
) {
	pkg := contextPkg(request)

	fmt.Fprintln(writer, pkg.Status)
}

func (web *Web) handlePackageQueue(
	writer http.ResponseWriter,
	request *http.Request,
) {
	pkg := contextPkg(request)

	err := web.collection.Update(
		bson.M{"name": pkg.Name},
		bson.M{"$set": bson.M{"status": StatusQueued}},
	)
	if err != nil {
		web.Error(writer, err)
		return
	}

	fmt.Fprintf(
		writer,
		"package %v has been added to the queue",
		pkg.Name,
	)
}

func (web *Web) handlePackageLogs(
	writer http.ResponseWriter,
	request *http.Request,
) {
	pkg := contextPkg(request)

	contents, err := ioutil.ReadFile(
		filepath.Join(web.config.LogsDir, pkg.Name),
	)
	if err != nil {
		web.Error(writer, err)
		return
	}

	fmt.Fprintln(writer, string(contents))
}

func (web *Web) Error(writer http.ResponseWriter, err error) {
	writer.WriteHeader(http.StatusInternalServerError)
}

func contextPkg(request *http.Request) *pkg {
	return request.Context().Value("pkg").(*pkg)
}
