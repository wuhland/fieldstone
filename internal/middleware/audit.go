package middleware

import (
	"net/http"
	"time"
)

type AuditEvent struct {
	RequestID  string
	Method     string
	Path       string
	Status     int
	UserID     string
	OccurredAt time.Time
}

// AuditEmit returns a middleware that sends audit events to ch after each request.
// The channel must be buffered; if full the event is dropped without blocking.
func AuditEmit(ch chan<- AuditEvent) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			event := AuditEvent{
				RequestID:  GetRequestID(r.Context()),
				Method:     r.Method,
				Path:       r.URL.Path,
				Status:     rw.status,
				OccurredAt: time.Now(),
			}
			select {
			case ch <- event:
			default:
			}
		})
	}
}
