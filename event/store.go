package event

import "github.com/distribution/distribution/v3/notifications"

type Store interface {
	WriteEvents(events []notifications.Event) error
	ReadEvents(offsetID string, limit int) ([]notifications.Event, error)
}
