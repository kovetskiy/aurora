package bus

import "encoding/json"
import "github.com/streadway/amqp"

type Delivery struct {
	amqp.Delivery
}

func (delivery *Delivery) Ack() error {
	return delivery.Delivery.Ack(false) // false for no multiple acks
}

func (delivery *Delivery) Reject() error {
	return delivery.Delivery.Reject(false) // false for requeue
}

func (delivery *Delivery) GetBody() []byte {
	return delivery.Body
}

func (delivery *Delivery) Decode(resource interface{}) error {
	err := json.Unmarshal(delivery.GetBody(), &resource)
	if err != nil {
		return err
	}

	return nil
}
