package nop

import "github.com/evanebb/regnotify/event"

type EventStore struct{}

func NewEventStore() EventStore {
	return EventStore{}
}

func (s EventStore) WriteEvents(events []event.Event) error {
	return nil
}

func (s EventStore) ReadEvents(keyOffset string, limit int) ([]event.Event, error) {
	return make([]event.Event, 0), nil
}
