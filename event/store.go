package event

import (
	"github.com/distribution/distribution/v3/notifications"
	"time"
)

type Store interface {
	WriteEvents(events []notifications.Event) error
	ReadEvents(offsetID string, limit int, from time.Time, until time.Time) ([]notifications.Event, error)
}
