package bus

import (
	"github.com/reconquest/karma-go"
	"github.com/streadway/amqp"
)

type Consumer struct {
	deliveries <-chan amqp.Delivery
	done       chan struct{}
}

func (bus *Bus) GetConsumer(queue string) (*Consumer, error) {
	deliveries, err := bus.channel.Consume(
		queue,
		"",    // auto-generated consumer tag
		false, // no automatic acks
		false, // no exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to consume queue",
		)
	}

	return &Consumer{
		deliveries: deliveries,
		done:       make(chan struct{}),
	}, nil
}

func (consumer *Consumer) Consume() (*Delivery, bool) {
	select {
	case <-consumer.done:
		return nil, false

	case delivery := <-consumer.deliveries:
		return &Delivery{delivery}, true
	}
}

func (consumer *Consumer) Close() error {
	close(consumer.done)
	return nil
}
