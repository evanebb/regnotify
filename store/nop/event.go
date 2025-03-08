package nop

import (
	"github.com/distribution/distribution/v3/notifications"
	"time"
)

type EventStore struct{}

func NewEventStore() EventStore {
	return EventStore{}
}

func (s EventStore) WriteEvents(events []notifications.Event) error {
	return nil
}

func (s EventStore) ReadEvents(keyOffset string, limit int, from time.Time, until time.Time) ([]notifications.Event, error) {
	return make([]notifications.Event, 0), nil
}
