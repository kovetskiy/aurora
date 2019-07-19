package bus

import (
	"github.com/reconquest/karma-go"
	"github.com/streadway/amqp"
)

type Consumer interface {
	Consume() (*Delivery, bool)
}

type QueueConsumer struct {
	deliveries <-chan amqp.Delivery
	done       chan struct{}
}

type ExchangeConsumer struct {
	QueueConsumer
}

func (ch *Channel) GetQueueConsumer(queue string) (*QueueConsumer, error) {
	deliveries, err := ch.channel.Consume(
		queue,
		"",    // auto-generated consumer tag
		false, // automatic acks
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	return &QueueConsumer{
		deliveries: deliveries,
		done:       make(chan struct{}),
	}, nil
}

func (ch *Channel) GetExchangeConsumer(exchangeName string, identitiy string) (*ExchangeConsumer, error) {
	queueName := exchangeName + ":" + identitiy

	_, err := ch.channel.QueueDeclare(
		queueName,
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to declare queue for exchange",
		)
	}

	routingKey := ""
	err = ch.channel.QueueBind(queueName, routingKey, exchangeName, false, nil)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to bind to exchange",
		)
	}

	deliveries, err := ch.channel.Consume(
		queueName,
		"",    // auto-generated consumer tag
		false, // automatic acks
		true,  // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	return &ExchangeConsumer{
		QueueConsumer{
			deliveries: deliveries,
			done:       make(chan struct{}),
		},
	}, nil
}

func (consumer *QueueConsumer) Consume() (*Delivery, bool) {
	select {
	case <-consumer.done:
		return nil, false

	case delivery := <-consumer.deliveries:
		return &Delivery{delivery}, true
	}
}

func (consumer *QueueConsumer) Close() error {
	close(consumer.done)
	return nil
}
