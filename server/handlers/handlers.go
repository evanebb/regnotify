package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/distribution/distribution/v3/notifications"
	"github.com/evanebb/regnotify/broker"
	"github.com/evanebb/regnotify/event"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type eventEnvelope struct {
	Events []notifications.Event `json:"events"`
}

func WriteEvents(logger *slog.Logger, store event.Store, broker *broker.Broker[notifications.Event]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var envelope eventEnvelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, "invalid JSON body given")
			return
		}

		if err := store.WriteEvents(envelope.Events); err != nil {
			logger.Error("failed to write events", "error", err)
			writeJSONResponse(w, http.StatusInternalServerError, "failed to write events")
			return
		}

		for _, e := range envelope.Events {
			broker.Publish(e)
		}

		writeJSONResponse(w, http.StatusOK, "successfully wrote events")
	}
}

func ReadEvents(logger *slog.Logger, store event.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")

		var limit int
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			var err error
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				writeJSONResponse(w, http.StatusBadRequest, "invalid 'limit' parameter given")
				return
			}
		}

		var from time.Time
		fromStr := r.URL.Query().Get("from")
		if fromStr != "" {
			var err error
			from, err = time.Parse(time.RFC3339, fromStr)
			if err != nil {
				writeJSONResponse(w, http.StatusBadRequest, "invalid 'from' parameter given, must be in RFC3339 format")
				return
			}
		}

		var until time.Time
		untilStr := r.URL.Query().Get("until")
		if untilStr != "" {
			var err error
			until, err = time.Parse(time.RFC3339, untilStr)
			if err != nil {
				writeJSONResponse(w, http.StatusBadRequest, "invalid 'until' parameter given, must be in RFC3339 format")
				return
			}
		}

		events, err := store.ReadEvents(offset, limit, from, until)
		if err != nil {
			logger.Error("failed to read events", "error", err)
			writeJSONResponse(w, http.StatusInternalServerError, "failed to read events")
			return
		}

		response := eventEnvelope{Events: events}
		writeJSONResponse(w, http.StatusOK, response)
	}
}

func WatchEvents(logger *slog.Logger, broker *broker.Broker[notifications.Event]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FIXME: is this necessary?
		w.Header().Set("Access-Control-Allow-Origin", "*")

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		clientDisconnect := r.Context().Done()

		rc := http.NewResponseController(w)

		ch := make(chan notifications.Event)
		broker.Subscribe(ch)

		for {
			select {
			case <-clientDisconnect:
				broker.Unsubscribe(ch)
				return
			case e := <-ch:
				encoded, err := json.Marshal(e)
				if err != nil {
					logger.Error("failed to encode event", "error", err)
					return
				}

				if _, err := fmt.Fprintf(w, "data: %s\n\n", encoded); err != nil {
					logger.Error("failed to write event to client", "error", err)
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
