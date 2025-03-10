package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
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
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body given")
			return
		}

		if err := store.WriteEvents(envelope.Events); err != nil {
			logger.Error("failed to write events", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to write events")
			return
		}

		for _, e := range envelope.Events {
			broker.Publish(e)
		}

		writeJSONSuccess(w, http.StatusOK, "successfully wrote events")
	}
}

func ReadEvents(logger *slog.Logger, store event.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := buildEventFilter(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		events, err := store.ReadEvents(filter)
		if err != nil {
			logger.Error("failed to read events", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to read events")
			return
		}

		response := eventEnvelope{Events: events}
		writeJSONSuccess(w, http.StatusOK, response)
	}
}

func buildEventFilter(r *http.Request) (event.Filter, error) {
	var filter event.Filter

	filter.OffsetID = r.URL.Query().Get("offset")

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		var err error
		filter.Limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return event.Filter{}, errors.New("invalid 'limit' parameter, must be an integer")
		}
	}

	fromStr := r.URL.Query().Get("from")
	if fromStr != "" {
		var err error
		filter.From, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return event.Filter{}, errors.New("invalid 'from' parameter, must be a valid RC3339 timestamp")
		}
	}

	untilStr := r.URL.Query().Get("until")
	if untilStr != "" {
		var err error
		filter.Until, err = time.Parse(time.RFC3339, untilStr)
		if err != nil {
			return event.Filter{}, errors.New("invalid 'until' parameter, must be a valid RC3339 timestamp")
		}
	}

	filter.SearchQuery = r.URL.Query().Get("searchQuery")

	return filter, nil
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

		filter, err := buildEventFilter(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		for {
			select {
			case <-clientDisconnect:
				broker.Unsubscribe(ch)
				return
			case e := <-ch:
				if !filter.From.IsZero() && filter.From.After(e.Timestamp) {
					// no sense in checking this, since the timestamp of new events should always be right now, but
					// just check it
					continue
				}

				if !filter.Until.IsZero() && filter.Until.Before(e.Timestamp) {
					continue
				}

				encoded, err := json.Marshal(e)
				if err != nil {
					logger.Error("failed to encode event", "error", err)
					return
				}

				if filter.SearchQuery != "" && !bytes.Contains(encoded, []byte(filter.SearchQuery)) {
					continue
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
