package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/evanebb/regnotify/broker"
	"github.com/evanebb/regnotify/configuration"
	"github.com/evanebb/regnotify/event"
	boltstore "github.com/evanebb/regnotify/store/bolt"
	"github.com/evanebb/regnotify/store/nop"
	bolt "go.etcd.io/bbolt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func Run(ctx context.Context, conf *configuration.Configuration) error {
	logger, err := buildLogger(conf)
	if err != nil {
		return err
	}

	var eventStore event.Store
	if conf.Storage.Bolt.Enabled {
		logger.Info("using boltdb as storage backend")
		db, err := bolt.Open(conf.Storage.Bolt.Path, 0o600, nil)
		if err != nil {
			return err
		}

		defer func(db *bolt.DB) {
			if err := db.Close(); err != nil {
				logger.Error("failed to close database", "error", err)
			}
		}(db)

		eventStore, err = boltstore.NewEventStore(db)
		if err != nil {
			return err
		}
	}

	if eventStore == nil {
		// if no storage backend is enabled, we do not store events, but just broadcast them to clients
		logger.Info("no storage backend configured, events will not be stored")
		eventStore = nop.NewEventStore()
	}

	eventBroker := broker.New[event.Event]()
	go eventBroker.Start()

	mux := http.NewServeMux()
	addRoutes(mux, logger, eventStore, eventBroker)

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}

	go func() {
		if conf.HTTP.Certificate != "" && conf.HTTP.Key != "" {
			logger.Info(fmt.Sprintf("starting https server on %s", server.Addr))
			if err := server.ListenAndServeTLS(conf.HTTP.Certificate, conf.HTTP.Key); !errors.Is(err, http.ErrServerClosed) {
				logger.Error("error listening and serving", "error", err)
			}
		} else {
			logger.Info(fmt.Sprintf("starting http server on %s", server.Addr))
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				logger.Error("error listening and serving", "error", err)
			}
		}
	}()

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

var logLevelMap = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func buildLogger(conf *configuration.Configuration) (*slog.Logger, error) {
	level, ok := logLevelMap[conf.Log.Level]
	if !ok {
		return nil, fmt.Errorf("invalid log level %q given", conf.Log.Level)
	}

	logHandlerOptions := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch conf.Log.Formatter {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, logHandlerOptions)
	case "text":
		handler = slog.NewTextHandler(os.Stderr, logHandlerOptions)
	}

	return slog.New(handler), nil
}
