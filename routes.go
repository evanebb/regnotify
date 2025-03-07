package main

import (
	"github.com/evanebb/regnotify/ui"
	bolt "go.etcd.io/bbolt"
	"log/slog"
	"net/http"
)

func addRoutes(mux *http.ServeMux, logger *slog.Logger, db *bolt.DB, eventBroker *EventBroker) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
	})

	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.FS(ui.Files))))

	mux.Handle("POST /api/v1/notifications", HandleWriteNotifications(logger, db, eventBroker))
	mux.Handle("GET /api/v1/notifications", HandleReadNotifications(logger, db))
	mux.Handle("GET /api/v1/notifications/watch", HandleWatchNotifications(logger, eventBroker))
}
