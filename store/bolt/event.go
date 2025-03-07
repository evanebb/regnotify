package bolt

import (
	"encoding/json"
	"github.com/evanebb/regnotify/event"
	bolt "go.etcd.io/bbolt"
	"time"
)

type EventStore struct {
	db *bolt.DB
}

func NewEventStore(db *bolt.DB) (EventStore, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("events"))
		return err
	})
	if err != nil {
		return EventStore{}, err
	}

	return EventStore{db: db}, nil
}

func (s EventStore) WriteEvents(events []event.Event) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("events"))

		for _, e := range events {
			// timestamp + ID is the key, so events are stored in chronological order
			e.Key = e.Timestamp.Format(time.RFC3339) + e.ID

			encoded, err := json.Marshal(e)
			if err != nil {
				return err
			}

			if err := b.Put([]byte(e.Key), encoded); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s EventStore) ReadEvents(keyOffset string, limit int) ([]event.Event, error) {
	events := make([]event.Event, 0)

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("events"))

		c := b.Cursor()

		var k, v []byte
		if keyOffset != "" {
			// if an offset is specified, start from it
			c.Seek([]byte(keyOffset))
			k, v = c.Prev()
		} else {
			// read values in reverse order, so we get the 'newest' values first
			k, v = c.Last()
		}

		for ; k != nil; k, v = c.Prev() {
			if limit > 0 && len(events) >= limit {
				break
			}

			var e event.Event
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}

			// always ensure that the key is up-to-date in the event
			e.Key = string(k)

			events = append(events, e)
		}

		return nil
	})

	return events, err
}
