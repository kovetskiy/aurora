package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/kovetskiy/aurora/pkg/bus"
	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/reconquest/karma-go"
)

func handleWatch(opts Options) error {
	client := NewClient(opts.Address)

	var response proto.ResponseGetBus
	err := client.Call(
		(*rpc.PackageService).GetBus,
		proto.RequestGetBus{
			Name: opts.Package,
		},
		&response,
	)
	if err != nil {
		return err
	}

	stream := response.Stream

	connection, _, err := websocket.DefaultDialer.Dial(
		stream, nil,
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to connect to logs stream: %s", stream,
		)
	}

	defer connection.Close()

	log.Printf("connected to logs stream: %s", stream)

	var message bus.Message
	for {
		_, reader, err := connection.NextReader()
		if err != nil {
			return err
		}

		err = json.NewDecoder(reader).Decode(&message)
		if err != nil {
			return err
		}

		switch message.Type {
		case "status":
			fmt.Printf("Status: %s\n", message.Data)
		case "log":
			fmt.Print(message.Data)
		default:
			log.Println("unhandled type of message: %q", message.Type)
		}
	}

	return nil
}