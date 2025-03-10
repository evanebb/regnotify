package bolt

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/distribution/distribution/v3/notifications"
	"github.com/evanebb/regnotify/event"
	bolt "go.etcd.io/bbolt"
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

// i64tob returns an 8-byte big endian representation of v.
// Credit to https://github.com/golang/go/issues/19635#issuecomment-384448277
func i64tob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (s EventStore) WriteEvents(events []notifications.Event) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		eventBucket := tx.Bucket([]byte("events"))
		eventIndexBucket := tx.Bucket([]byte("events_id_index"))

		for _, e := range events {
			// timestamp + ID is the key, so events are stored in chronological order while still having a unique key
			unixTime := e.Timestamp.UTC().UnixNano()
			key := append(i64tob(unixTime), []byte(e.ID)...)

			encoded, err := json.Marshal(e)
			if err != nil {
				return err
			}

			if err := eventBucket.Put(key, encoded); err != nil {
				return err
			}

			if err := eventIndexBucket.Put([]byte(e.ID), key); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s EventStore) ReadEvents(filter event.Filter) ([]notifications.Event, error) {
	events := make([]notifications.Event, 0)

	err := s.db.View(func(tx *bolt.Tx) error {
		eventBucket := tx.Bucket([]byte("events"))
		eventIndexBucket := tx.Bucket([]byte("events_id_index"))

		c := eventBucket.Cursor()

		k, v := c.Last()
		if !filter.Until.IsZero() {
			c.Seek(i64tob(filter.Until.UTC().UnixNano()))
			// always go back one in case the exact key doesn't exist, so we do not risk grabbing an event after the
			// until date
			k, v = c.Prev()
		}

		if filter.OffsetID != "" {
			// if an offset is specified, start from it
			// use the event ID index to get the key for the event
			offsetKey := eventIndexBucket.Get([]byte(filter.OffsetID))

			keyComparison := bytes.Compare(offsetKey, k)
			if keyComparison < 0 {
				// we should only start from the offset ID if its key is further down in the bucket than the current key
				c.Seek(offsetKey)
			}
			if keyComparison <= 0 {
				// go back one item, since we don't want to include the item with the offset ID key itself
				// also do this if the offset is the current key
				k, v = c.Prev()
			}
		}

		var fromBytes []byte
		if !filter.From.IsZero() {
			fromBytes = i64tob(filter.From.UTC().UnixNano())
		}

		var searchQueryBytes []byte
		if filter.SearchQuery != "" {
			searchQueryBytes = []byte(filter.SearchQuery)
		}

		// read values in reverse order, so we get the newest values first
		for ; k != nil; k, v = c.Prev() {
			if filter.Limit > 0 && len(events) >= filter.Limit {
				break
			}

			// if we are given a from date, read until we reach it
			if fromBytes != nil && bytes.Compare(k, fromBytes) <= 0 {
				break
			}

			// brute-force search the JSON for the given query
			// note: this isn't ideal and can be very slow if we have a lot of events. filtering on specific (indexed)
			// fields like the repository name or implementing some kind of proper full-text search would be better, but
			// it works fine for now, especially if you 'pre-filter' it a bit using a time range
			if searchQueryBytes != nil && !bytes.Contains(v, searchQueryBytes) {
				continue
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
