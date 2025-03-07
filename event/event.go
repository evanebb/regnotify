package event

import "github.com/distribution/distribution/v3/notifications"

type Event struct {
	Key string `json:"key"`
	notifications.Event
}

type Store interface {
	WriteEvents(events []Event) error
	ReadEvents(keyOffset string, limit int) ([]Event, error)
}
