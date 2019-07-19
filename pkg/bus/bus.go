package bus

import (
	"github.com/reconquest/karma-go"
	"github.com/streadway/amqp"
)

// Need to have const list of message types

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

type Bus struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

func Dial(uri string) (*Bus, error) {
	connection, err := amqp.Dial(uri)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to dial to amqp",
		)
	}

	channel, err := connection.Channel()
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to get aqmp channel",
		)
	}

	bus := &Bus{}
	bus.connection = connection
	bus.channel = channel

	return bus, nil
}

// Close current connection and opened channels.
func (bus *Bus) Close() error {
	if bus.connection == nil {
		return nil
	}

	err := bus.connection.Close()
	if err != nil {
		return err
	}

	return nil
}
