package main

import (
	"io"
	"log"
	"os"

	"github.com/gorilla/websocket"
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

	for {
		_, reader, err := connection.NextReader()
		if err != nil {
			return err
		}

		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			return err
		}
	}

	return nil
}
