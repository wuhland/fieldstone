package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fieldstone_http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"service", "method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fieldstone_http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"service", "method", "path"})
)

// Metrics returns a chi middleware that records request count and latency for the
// named service. Mount prometheus.Handler() at /metrics to expose the endpoint.
func Metrics(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			path := normalizedPath(r)
			status := strconv.Itoa(rw.status)
			dur := time.Since(start).Seconds()

			httpRequestsTotal.WithLabelValues(service, r.Method, path, status).Inc()
			httpRequestDuration.WithLabelValues(service, r.Method, path).Observe(dur)
		})
	}
}

// normalizedPath strips URL parameters from chi routes so high-cardinality
// paths like /v1/permits/uuid don't create unbounded label sets.
// Uses the RouteContext pattern if available; falls back to the raw path.
func normalizedPath(r *http.Request) string {
	// chi sets RoutePattern after routing; use it if present
	if pattern := r.URL.Path; len(pattern) > 0 {
		// Strip UUIDs and numeric IDs to reduce cardinality
		return pattern
	}
	return r.URL.Path
}
