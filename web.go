package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type webserver struct {
	database *database
}

func serveWeb(address string, db *database) error {
	gin.SetMode(gin.ReleaseMode)

	webserver := &webserver{
		database: db,
	}

	router := gin.New()
	router.Handle(
		"GET", "/v1/packages/", webserver.handlePackages,
	)

	return nil
}

func (webserver *webserver) handlePackages(context *gin.Context) {
	err := webserver.database.sync()
	if err != nil {
		context.IndentedJSON(
			http.StatusInternalServerError,
			gin.H{"error": err.Error()},
		)
		return
	}

	context.IndentedJSON(http.StatusOK, webserver.database.getData())
}
