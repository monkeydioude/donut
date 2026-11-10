package corehttp

import (
	"log/slog"
	"net/http"
	"time"
)

// LogRequests logs every request at debug level: when it starts, then its status and duration.
// Only the path is logged, never the query string: the websocket's carries the player's secret guest id.
func LogRequests(log *slog.Logger) func(http.Handler) http.Handler {
	log = log.With("component", "http")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			attrs := []any{"method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "origin", r.Header.Get("Origin")}
			log.DebugContext(r.Context(), "http request", attrs...)
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			log.DebugContext(r.Context(), "http done", append(attrs, "status", recorder.status, "bytes", recorder.bytes, "took", time.Since(start))...)
		})
	}
}

// statusRecorder notes the status and size of a response.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// Unwrap lets the websocket upgrade reach the real ResponseWriter to take the connection over.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
