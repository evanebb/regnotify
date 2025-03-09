package nop

import (
	"github.com/distribution/distribution/v3/notifications"
	"github.com/evanebb/regnotify/event"
)

type EventStore struct{}

func NewEventStore() EventStore {
	return EventStore{}
}

func (s EventStore) WriteEvents(events []notifications.Event) error {
	return nil
}

func (s EventStore) ReadEvents(filter event.Filter) ([]notifications.Event, error) {
	return make([]notifications.Event, 0), nil
}
