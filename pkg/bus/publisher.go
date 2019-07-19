package bus

import (
	"encoding/json"

	"github.com/reconquest/karma-go"
	"github.com/streadway/amqp"
)

type Publisher interface {
	Publish(interface{}) error
}

type QueuePublisher struct {
	queue   *amqp.Queue
	channel *amqp.Channel
}

type ExchangePublisher struct {
	queueName string
	channel   *amqp.Channel
}

func (ch *Channel) GetQueuePublisher(queueName string) (*QueuePublisher, error) {
	queue, err := ch.channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	return &QueuePublisher{
		queue:   &queue,
		channel: ch.channel,
	}, nil
}

func (publisher *QueuePublisher) Publish(message interface{}) error {
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

func (ch *Channel) GetExchangePublisher(queueName string) (*ExchangePublisher, error) {
	err := ch.channel.ExchangeDeclare(
		queueName,
		"fanout", // kind
		false,    // durable
		true,     // delete when unused
		false,    // internal
		false,    // noWait
		nil,      // args
	)
	if err != nil {
		return nil, err
	}

	// enter confirming mode so we sure that server received the message
	err = ch.channel.Confirm(false)
	if err != nil {
		return nil, err
	}

	return &ExchangePublisher{
		channel:   ch.channel,
		queueName: queueName,
	}, nil
}

func (publisher *ExchangePublisher) Publish(message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return karma.Format(
			err,
			"unable to marshal message",
		)
	}

	err = publisher.channel.Publish(
		publisher.queueName,
		"",
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
