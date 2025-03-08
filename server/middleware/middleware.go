package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type ResponseWriterWrapper interface {
	http.ResponseWriter
	Status() int
}

func NewResponseWriterWrapper(w http.ResponseWriter) ResponseWriterWrapper {
	_, flushable := w.(http.Flusher)

	basic := basicWriterWrapper{ResponseWriter: w}

	if flushable {
		return &flushableWriterWrapper{basicWriterWrapper: basic}
	}

	return &basic
}

type basicWriterWrapper struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *basicWriterWrapper) Status() int {
	return w.status
}

func (w *basicWriterWrapper) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *basicWriterWrapper) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(data)
}

type flushableWriterWrapper struct {
	basicWriterWrapper
}

func (w *flushableWriterWrapper) Flush() {
	w.wroteHeader = true
	f := w.basicWriterWrapper.ResponseWriter.(http.Flusher)
	f.Flush()
}

// Logger will log information about the HTTP request that was made.
func Logger(l *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := NewResponseWriterWrapper(w)
			next.ServeHTTP(wrapped, r)

			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}

			duration := time.Since(start)

			l.Info(
				fmt.Sprintf("%s %s://%s%s from %s - %d in %s",
					r.Method,
					scheme,
					r.Host,
					r.URL.EscapedPath(),
					r.RemoteAddr,
					wrapped.Status(),
					duration,
				),
				"method", r.Method,
				"scheme", scheme,
				"host", r.Host,
				"path", r.URL.EscapedPath(),
				"remoteAddress", r.RemoteAddr,
				"status", wrapped.Status(),
				"duration", duration,
			)
		}
		return http.HandlerFunc(fn)
	}
}
