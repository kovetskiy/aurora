package bus

import (
	"encoding/json"
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

func TestBus_Direct_TwoPublishersOneConsumer(t *testing.T) {
	test := assert.New(t)

	bus, err := Dial(envBusAddress)
	if err != nil {
		panic(err.Error())
	}

	queueName := "go-test-" + strconv.Itoa(rand.Int())
	msg := "message"

	publisher1, err := bus.GetPublisher(queueName)
	test.NoError(err)

	err = publisher1.Publish(msg)
	test.NoError(err)

	publisher2, err := bus.GetPublisher(queueName)
	test.NoError(err)

	err = publisher2.Publish(msg)
	test.NoError(err)

	consumer1, err := bus.GetConsumer(queueName)
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

func decode(data []byte, resource interface{}) {
	err := json.Unmarshal(data, resource)
	if err != nil {
		panic(karma.Format(err, "unable to decode: %s", string(data)))
	}
}
