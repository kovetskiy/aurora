package main

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// WIP

type MessageStatus struct {
	//
}

func (web *Web) handlePackageLogsFollow(
	response http.ResponseWriter,
	request *http.Request,
) {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1,
		WriteBufferSize: 1,
	}

	conn, err := upgrader.Upgrade(response, request, nil)
	if err != nil {
		web.Error(response, err)
		return
	}

	defer conn.Close()

	writer, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		web.Error(response, err)
		return
	}

	_ = writer
}
