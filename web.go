package main

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type webserver struct {
	collection *mgo.Collection
	logsDir    string
}

func serveWeb(collection *mgo.Collection, address, repository, logsDir string) error {
	webserver := &webserver{
		collection: collection,
		logsDir:    logsDir,
	}

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(getRouterRecovery(), getRouterLogger())

	router.Static("/aurora", repository)

	router.
		Handle(
			"GET", "/api/v1/pkg/", webserver.handlePackagesList,
		).
		Handle(
			"GET", "/api/v1/pkg/:name", webserver.handlePackageInformation,
		)

	return router.Run(address)
}

func (webserver *webserver) handlePackagesList(context *gin.Context) {
	packages := []bson.M{}
	webserver.collection.Find(bson.M{}).All(&packages)
	context.IndentedJSON(http.StatusOK, packages)
}

func (webserver *webserver) handlePackageInformation(context *gin.Context) {
	pkg := &pkg{}

	webserver.collection.Find(bson.M{"name": context.Param("name")}).One(&pkg)

	contents, err := ioutil.ReadFile(
		filepath.Join(webserver.logsDir, pkg.Name),
	)
	if err != nil {
		context.IndentedJSON(
			http.StatusInternalServerError,
			gin.H{"error": err.Error()},
		)
		return
	}

	context.Data(http.StatusOK, "text/plain", contents)
}

func getRouterRecovery() gin.HandlerFunc {
	return func(context *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := getStack(3)
				errorf("PANIC: %s\n%s", err, stack)

				context.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}()

		context.Next()
	}
}

func getRouterLogger() gin.HandlerFunc {
	return func(context *gin.Context) {
		start := time.Now()

		// Process request
		context.Next()

		duration := time.Now().Sub(start)

		infof(
			"%v %-4v %v %v %v",
			context.ClientIP(),
			context.Request.Method,
			context.Request.RequestURI,
			context.Writer.Status(),
			duration,
		)
	}
}

func getStack(skip int) string {
	buffer := make([]byte, 1024)
	for {
		written := runtime.Stack(buffer, true)
		if written < len(buffer) {
			// call stack contains of goroutine number and set of calls
			//   goroutine NN [running]:
			//   github.com/user/project.(*Type).MethodFoo()
			//        path/to/src.go:line
			//   github.com/user/project.MethodBar()
			//        path/to/src.go:line
			// so if we need to skip 2 calls than we must split stack on
			// following parts:
			//   2(call)+2(call path)+1(goroutine header) + 1(callstack)
			// and extract first and last parts of resulting slice
			stack := strings.SplitN(string(buffer[:written]), "\n", skip*2+2)
			return stack[0] + "\n" + stack[skip*2+1]
		}

		buffer = make([]byte, 2*len(buffer))
	}
}
