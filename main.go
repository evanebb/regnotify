package main

import (
	"context"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func run(ctx context.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	db, err := bolt.Open("notifications.db", 0o600, nil)
	if err != nil {
		return err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("notifications"))
		return err
	})
	if err != nil {
		return err
	}

	defer func(db *bolt.DB) {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}(db)

	eventBroker := NewEventBroker()
	go eventBroker.Start()

	handler := NewServer(logger, db, eventBroker)

	server := &http.Server{
		Addr:    ":8000",
		Handler: handler,
	}

	logger.Info("listening on :8000")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("error listening and serving", "error", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("shutting down http server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("error shutting down http server: %w", err)
	}

	logger.Info("shutting down event broker")
	eventBroker.Stop()

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
