package bus

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/reconquest/karma-go"
	"github.com/stretchr/testify/assert"
)

var (
	envBusAddress = os.Getenv("TEST_BUS_ADDR")
)

func getQueueName() string {
	return "go-test-" + strconv.Itoa(rand.Int())
}

func TestBus_Queue_TwoPublishersOneConsumer(t *testing.T) {
	test := assert.New(t)

	bus, err := Dial(envBusAddress)
	if err != nil {
		panic(err.Error())
	}

	queueName := getQueueName()
	msg := "message"

	ch1, err := bus.Channel()
	test.NoError(err)

	publisher1, err := ch1.GetQueuePublisher(queueName)
	test.NoError(err)

	err = publisher1.Publish(msg)
	test.NoError(err)

	ch2, err := bus.Channel()
	test.NoError(err)

	publisher2, err := ch2.GetQueuePublisher(queueName)
	test.NoError(err)

	err = publisher2.Publish(msg)
	test.NoError(err)

	ch3, err := bus.Channel()
	test.NoError(err)

	consumer1, err := ch3.GetQueueConsumer(queueName)
	test.NoError(err)

	delivery1, ok := consumer1.Consume()
	test.True(ok)

	var received1 string
	decode(delivery1.GetBody(), &received1)

	test.Equal(msg, received1)

	delivery2, ok := consumer1.Consume()
	test.True(ok)

	var received2 string
	decode(delivery2.GetBody(), &received2)

	test.Equal(msg, received2)

	_ = test
}

func TestBus_Exchange_PubSub(t *testing.T) {
	test := assert.New(t)

	bus, err := Dial(envBusAddress)
	if err != nil {
		panic(err.Error())
	}

	queueName := "go-test-" + strconv.Itoa(rand.Int())
	msg := "message"

	chPub, err := bus.Channel()
	test.NoError(err)

	publisher, err := chPub.GetExchangePublisher(queueName)
	test.NoError(err)

	var consumers []Consumer
	const maxConsumers = 3
	for i := 0; i < maxConsumers; i++ {
		ch, err := bus.Channel()
		test.NoError(err)

		identity := fmt.Sprintf("consumer-%d", i)
		consumer, err := ch.GetExchangeConsumer(queueName, identity)
		test.NoError(err)

		consumers = append(consumers, consumer)
	}

	err = publisher.Publish(msg)
	test.NoError(err)

	for _, consumer := range consumers {
		delivery, ok := consumer.Consume()
		test.True(ok)

		var got string
		decode(delivery.GetBody(), &got)

		test.Equal(msg, got)
	}
}

func decode(data []byte, resource interface{}) {
	err := json.Unmarshal(data, resource)
	if err != nil {
		panic(karma.Format(err, "unable to decode: %s", string(data)))
	}
}
