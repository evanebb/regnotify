package main

import (
	"encoding/json"
	"fmt"
	"github.com/distribution/distribution/v3/notifications"
	bolt "go.etcd.io/bbolt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type RequestEnvelope struct {
	Events []notifications.Event `json:"events"`
}

type ResponseEnvelope struct {
	Events []ExtendedEvent `json:"events"`
}

type ExtendedEvent struct {
	Key string `json:"key"`
	notifications.Event
}

func HandleWriteNotifications(logger *slog.Logger, db *bolt.DB, eventBroker *EventBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var envelope RequestEnvelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, "invalid JSON body given")
			return
		}

		err := db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("notifications"))
			for _, event := range envelope.Events {
				// timestamp + ID is the key, so events are stored in chronological order
				id := event.Timestamp.Format(time.RFC3339) + event.ID

				encoded, err := json.Marshal(event)
				if err != nil {
					return err
				}

				if err := b.Put([]byte(id), encoded); err != nil {
					return err
				}

				eventBroker.Publish(event)
			}

			return nil
		})
		if err != nil {
			logger.Error("failed to write notifications", "error", err)
			writeJSONResponse(w, http.StatusInternalServerError, "failed to write notifications")
			return
		}

		writeJSONResponse(w, http.StatusOK, "successfully wrote notifications")
	}
}

func HandleReadNotifications(logger *slog.Logger, db *bolt.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		events := make([]ExtendedEvent, 0)

		lastQuery := r.URL.Query().Get("last")

		var size int
		sizeQuery := r.URL.Query().Get("size")
		if sizeQuery != "" {
			size, err = strconv.Atoi(sizeQuery)
			if err != nil {
				writeJSONResponse(w, http.StatusBadRequest, "invalid 'size' parameter given")
				return
			}
		}

		err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("notifications"))

			c := b.Cursor()

			var k, v []byte
			if lastQuery != "" {
				c.Seek([]byte(lastQuery))
				k, v = c.Prev()
			} else {
				// read values in reverse order, so we get the 'newest' values first
				k, v = c.Last()
			}

			for ; k != nil; k, v = c.Prev() {
				if size > 0 && len(events) >= size {
					break
				}

				var event ExtendedEvent
				if err := json.Unmarshal(v, &event); err != nil {
					return err
				}

				event.Key = string(k)

				events = append(events, event)
			}

			return nil
		})
		if err != nil {
			logger.Error("failed to read notifications", "error", err)
			writeJSONResponse(w, http.StatusInternalServerError, "failed to read notifications")
			return
		}

		response := ResponseEnvelope{Events: events}
		writeJSONResponse(w, http.StatusOK, response)
	}
}

func HandleWatchNotifications(logger *slog.Logger, eventBroker *EventBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FIXME: is this necessary?
		w.Header().Set("Access-Control-Allow-Origin", "*")

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		clientDisconnect := r.Context().Done()

		rc := http.NewResponseController(w)

		ch := make(chan notifications.Event)
		eventBroker.Subscribe(ch)

		for {
			select {
			case <-clientDisconnect:
				eventBroker.Unsubscribe(ch)
				return
			case event := <-ch:
				encoded, err := json.Marshal(event)
				if err != nil {
					logger.Error("failed to encode event", "error", err)
					return
				}

				if _, err := fmt.Fprintf(w, "data: %s\n\n", encoded); err != nil {
					logger.Error("failed to write event", "error", err)
					return
				}

				if err := rc.Flush(); err != nil {
					logger.Error("failed to flush response", "error", err)
					return
				}
			}
		}
	}
}
