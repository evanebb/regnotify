package server

import (
	"github.com/distribution/distribution/v3/notifications"
	"github.com/evanebb/regnotify/broker"
	"github.com/evanebb/regnotify/event"
	"github.com/evanebb/regnotify/server/handlers"
	"github.com/evanebb/regnotify/ui"
	"log/slog"
	"net/http"
)

func addRoutes(mux *http.ServeMux, logger *slog.Logger, eventStore event.Store, eventBroker *broker.Broker[notifications.Event]) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
	})

	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.FS(ui.Files))))

	mux.Handle("POST /api/v1/events", handlers.WriteEvents(logger, eventStore, eventBroker))
	mux.Handle("GET /api/v1/events", handlers.ReadEvents(logger, eventStore))
	mux.Handle("GET /api/v1/events/watch", handlers.WatchEvents(logger, eventBroker))
}
