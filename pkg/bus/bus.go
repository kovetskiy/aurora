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

type Connection struct {
	connection *amqp.Connection
}

type Channel struct {
	channel *amqp.Channel
}

func Dial(uri string) (*Connection, error) {
	connection, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}

	return &Connection{connection: connection}, nil
}

func (conn *Connection) Channel() (*Channel, error) {
	channel, err := conn.connection.Channel()
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to get aqmp channel",
		)
	}

	return &Channel{channel: channel}, nil
}

// Close current connection and opened channels.
func (bus *Connection) Close() error {
	if bus.connection == nil {
		return nil
	}

	err := bus.connection.Close()
	if err != nil {
		return err
	}

	return nil
}
