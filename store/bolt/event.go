package bolt

import (
	"bytes"
	"encoding/json"
	"github.com/distribution/distribution/v3/notifications"
	bolt "go.etcd.io/bbolt"
	"time"
)

type EventStore struct {
	db *bolt.DB
}

func NewEventStore(db *bolt.DB) (EventStore, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("events")); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte("events_id_index")); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return EventStore{}, err
	}

	return EventStore{db: db}, nil
}

func (s EventStore) WriteEvents(events []notifications.Event) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		eventBucket := tx.Bucket([]byte("events"))
		eventIndexBucket := tx.Bucket([]byte("events_id_index"))

		for _, e := range events {
			// timestamp + ID is the key, so events are stored in chronological order while still having a unique key
			key := e.Timestamp.Format(time.RFC3339) + e.ID

			encoded, err := json.Marshal(e)
			if err != nil {
				return err
			}

			if err := eventBucket.Put([]byte(key), encoded); err != nil {
				return err
			}

			if err := eventIndexBucket.Put([]byte(e.ID), []byte(key)); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s EventStore) ReadEvents(offsetID string, limit int, from time.Time, until time.Time) ([]notifications.Event, error) {
	events := make([]notifications.Event, 0)

	err := s.db.View(func(tx *bolt.Tx) error {
		eventBucket := tx.Bucket([]byte("events"))
		eventIndexBucket := tx.Bucket([]byte("events_id_index"))

		c := eventBucket.Cursor()

		// read values in reverse order, so we get the 'newest' values first
		var k, v []byte
		if offsetID != "" {
			// if an offset is specified, start from it
			// use the event ID index to get the key for the event
			offsetKey := eventIndexBucket.Get([]byte(offsetID))
			c.Seek(offsetKey)
			// go back one item, since we don't want to include the item with the offset ID key itself
			k, v = c.Prev()
		} else {
			// if no offset is specified, just start at the end
			k, v = c.Last()
		}

		for ; k != nil; k, v = c.Prev() {
			if limit > 0 && len(events) >= limit {
				break
			}

			if !until.IsZero() && bytes.Compare(k, []byte(until.Format(time.RFC3339))) > 0 {
				// it would be better to use c.Seek() here, but every seek will reset the cursor to the beginning,
				// so it can't be combined with the offset ID seek
				continue
			}

			if !from.IsZero() && bytes.Compare(k, []byte(from.Format(time.RFC3339))) <= 0 {
				break
			}

			var e notifications.Event
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}

			events = append(events, e)
		}

		return nil
	})

	return events, err
}
