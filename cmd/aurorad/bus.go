package main

import (
	"sync"
)

type BusSubscription chan interface{}

type Bus struct {
	mutex  *sync.Mutex
	topics map[string]struct{}
	subs   map[string][]BusSubscription
}

func NewBus() *Bus {
	return &Bus{
		mutex:  &sync.Mutex{},
		topics: map[string]struct{}{},
		subs:   map[string][]BusSubscription{},
	}
}

// Subscribe returns channel subscribed on events.
func (bus *Bus) Subscribe(topic string) (BusSubscription, bool) {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	sub := make(chan interface{}, 1)

	bus.subs[topic] = append(bus.subs[topic], sub)

	_, ok := bus.topics[topic]

	return sub, ok
}

func (bus *Bus) Publish(topic string, data interface{}) {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()
	if _, ok := bus.topics[topic]; !ok {
		bus.topics[topic] = struct{}{}
	}

	for _, sub := range bus.subs[topic] {
		sub <- data
	}
}

func (bus *Bus) Close(topic string) {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	for len(bus.subs[topic]) > 0 {
		bus.unsubscribe(topic, bus.subs[topic][0])
	}

	delete(bus.topics, topic)
}

func (bus *Bus) Unsubscribe(topic string, sub BusSubscription) {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	bus.unsubscribe(topic, sub)
}

func (bus *Bus) unsubscribe(topic string, sub BusSubscription) {
	found := true
	for i := 0; i < len(bus.subs[topic]); i++ {
		if bus.subs[topic][i] == sub {
			bus.subs[topic] = append(
				bus.subs[topic][:i],
				bus.subs[topic][i+1:]...,
			)
			break
		}
	}
	if !found {
		return
	}

	close(sub)
}
