package event

import (
	"github.com/distribution/distribution/v3/notifications"
	"time"
)

type Filter struct {
	OffsetID string
	Limit    int
	From     time.Time
	Until    time.Time
}

type Store interface {
	WriteEvents(events []notifications.Event) error
	ReadEvents(filter Filter) ([]notifications.Event, error)
}
