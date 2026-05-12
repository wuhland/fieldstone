package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/fieldstone/fieldstone/internal/middleware"
)

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		now := time.Now()

		rl.mu.Lock()
		times := rl.requests[ip]
		var recent []time.Time
		for _, t := range times {
			if now.Sub(t) < rl.window {
				recent = append(recent, t)
			}
		}
		recent = append(recent, now)
		rl.requests[ip] = recent
		count := len(recent)
		rl.mu.Unlock()

		if count > rl.limit {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded","request_id":"` + middleware.GetRequestID(r.Context()) + `"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
