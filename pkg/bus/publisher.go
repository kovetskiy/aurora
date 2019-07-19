package bus

import (
	"encoding/json"

	"github.com/reconquest/karma-go"
	"github.com/streadway/amqp"
)

type Publisher struct {
	queue   *amqp.Queue
	channel *amqp.Channel
}

func (bus *Bus) declare(name string) (*amqp.Queue, error) {
	queue, err := bus.channel.QueueDeclare(
		name,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}

	return &queue, nil
}

func (bus *Bus) GetPublisher(queueName string) (*Publisher, error) {
	queue, err := bus.declare(queueName)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to declare queue",
		)
	}

	return &Publisher{
		queue:   queue,
		channel: bus.channel,
	}, nil
}

func (publisher *Publisher) Publish(message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return karma.Format(
			err,
			"unable to marshal message",
		)
	}

	err = publisher.channel.Publish(
		"",
		publisher.queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		},
	)
	if err != nil {
		return err
	}

	return nil
}
