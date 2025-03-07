package main

import (
	"github.com/distribution/distribution/v3/notifications"
)

type EventBroker struct {
	stopCh    chan struct{}
	publishCh chan notifications.Event
	subCh     chan chan<- notifications.Event
	unsubCh   chan chan<- notifications.Event
}

func NewEventBroker() *EventBroker {
	return &EventBroker{
		stopCh:    make(chan struct{}),
		publishCh: make(chan notifications.Event, 1),
		subCh:     make(chan chan<- notifications.Event, 1),
		unsubCh:   make(chan chan<- notifications.Event, 1),
	}
}

func (b *EventBroker) Start() {
	subscribers := make(map[chan<- notifications.Event]struct{})
	for {
		select {
		case <-b.stopCh:
			return
		case msgCh := <-b.subCh:
			subscribers[msgCh] = struct{}{}
		case msgCh := <-b.unsubCh:
			delete(subscribers, msgCh)
			close(msgCh)
		case event := <-b.publishCh:
			for msgCh := range subscribers {
				select {
				case msgCh <- event:
				default:
				}
			}
		}
	}
}

func (b *EventBroker) Stop() {
	close(b.stopCh)
}

func (b *EventBroker) Subscribe(ch chan<- notifications.Event) {
	b.subCh <- ch
}

func (b *EventBroker) Unsubscribe(ch chan<- notifications.Event) {
	b.unsubCh <- ch
}

func (b *EventBroker) Publish(event notifications.Event) {
	b.publishCh <- event
}
