package main

import (
	bolt "go.etcd.io/bbolt"
	"log/slog"
	"net/http"
)

func NewServer(logger *slog.Logger, db *bolt.DB, eventBroker *EventBroker) http.Handler {
	mux := http.NewServeMux()

	addRoutes(mux, logger, db, eventBroker)

	//loggerMiddleware := LoggerMiddleware(logger)
	//
	//return loggerMiddleware(mux)

	return mux
}
