package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/proto"
)

type BusServer struct {
	bus *Bus
}

func NewBusServer(bus *Bus) *BusServer {
	return &BusServer{
		bus: bus,
	}
}

func (server *BusServer) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	query := request.URL.Query()

	pkgName := query.Get("package")
	if pkgName == "" {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	sub, exists := server.bus.Subscribe(pkgName)

	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1,
		WriteBufferSize: 1,
	}

	connection, err := upgrader.Upgrade(response, request, nil)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	defer connection.Close()

	if !exists {
		err := connection.WriteJSON(bus.Message{
			Type: "empty_channel",
		})
		if err != nil {
			errorln(err)
			return
		}
	}

	for {
		message, ok := <-sub
		if !ok {
			tracef("sub for %s closed", pkgName)
			break
		}

		switch data := message.(type) {
		case proto.BuildStatus:
			err = connection.WriteJSON(bus.Message{
				Type: "status",
				Data: data.String(),
			})

		case string:
			err = connection.WriteJSON(bus.Message{
				Type: "log",
				Data: data,
			})

		default:
			panic(
				fmt.Errorf(
					"unknown type of message in bus: %T %#v",
					message,
					message,
				),
			)
		}

		if err != nil {
			errorln(err)
			server.bus.Unsubscribe(pkgName, sub)
			return
		}
	}

	server.bus.Unsubscribe(pkgName, sub)
}
