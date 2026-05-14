package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/fieldstone/fieldstone/internal/middleware"
)

// ─── Interface ────────────────────────────────────────────────────────────────

// RateLimiter checks whether a request identified by key is within the limit.
type RateLimiter interface {
	Allow(ctx context.Context, key string) bool
}

func rateLimitMiddleware(rl RateLimiter, limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if !rl.Allow(r.Context(), key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error":      "rate limit exceeded",
					"request_id": middleware.GetRequestID(r.Context()),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─── Memory implementation (single instance) ──────────────────────────────────

type memoryLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newMemoryLimiter(limit int, window time.Duration) *memoryLimiter {
	return &memoryLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (m *memoryLimiter) Allow(_ context.Context, key string) bool {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	times := m.requests[key]
	recent := times[:0]
	for _, t := range times {
		if now.Sub(t) < m.window {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	m.requests[key] = recent
	return len(recent) <= m.limit
}

// ─── Redis sliding-window implementation (distributed) ────────────────────────
//
// Uses a sorted set per IP. Each request adds a member with the current
// unix-nano timestamp as score. Members older than the window are pruned on
// every request. The count of remaining members determines whether the request
// is allowed. No external library — communicates via raw RESP over TCP.

type redisLimiter struct {
	addr   string
	limit  int
	window time.Duration
	pool   chan net.Conn
}

func newRedisLimiter(redisURL string, limit int, window time.Duration) (*redisLimiter, error) {
	addr := strings.TrimPrefix(redisURL, "redis://")
	// Probe the connection
	c, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	c.Close()

	rl := &redisLimiter{
		addr:   addr,
		limit:  limit,
		window: window,
		pool:   make(chan net.Conn, 16),
	}
	return rl, nil
}

func (r *redisLimiter) Allow(ctx context.Context, key string) bool {
	count, err := r.slidingWindow(ctx, "rl:"+key)
	if err != nil {
		slog.Error("redis rate limit error, falling back to allow", "error", err)
		return true // fail open
	}
	return count <= r.limit
}

func (r *redisLimiter) slidingWindow(ctx context.Context, key string) (int, error) {
	conn, err := r.acquire()
	if err != nil {
		return 0, err
	}
	defer r.release(conn)

	now := time.Now()
	windowStart := now.Add(-r.window).UnixNano()
	member := uuid.New().String()
	score := strconv.FormatInt(now.UnixNano(), 10)
	windowStr := strconv.FormatInt(windowStart, 10)
	expireSec := strconv.Itoa(int(r.window.Seconds()) + 1)

	// Pipeline: ZREMRANGEBYSCORE, ZADD, ZCARD, EXPIRE
	cmd := fmt.Sprintf(
		"*4\r\n$17\r\nZREMRANGEBYSCORE\r\n$%d\r\n%s\r\n$1\r\n0\r\n$%d\r\n%s\r\n"+
			"*4\r\n$4\r\nZADD\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n"+
			"*2\r\n$5\r\nZCARD\r\n$%d\r\n%s\r\n"+
			"*3\r\n$6\r\nEXPIRE\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
		len(key), key, len(windowStr), windowStr,
		len(key), key, len(score), score, len(member), member,
		len(key), key,
		len(key), key, len(expireSec), expireSec,
	)

	conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := fmt.Fprint(conn, cmd); err != nil {
		conn.Close()
		r.pool <- nil // invalidate slot
		return 0, fmt.Errorf("write: %w", err)
	}

	// Read 4 responses: ZREMRANGEBYSCORE, ZADD, ZCARD, EXPIRE
	br := bufio.NewReader(conn)
	for i := 0; i < 2; i++ { // skip ZREMRANGEBYSCORE and ZADD replies
		if _, err := readRESP(br); err != nil {
			return 0, err
		}
	}
	countVal, err := readRESP(br)
	if err != nil {
		return 0, err
	}
	if _, err := readRESP(br); err != nil { // EXPIRE reply
		return 0, err
	}

	count, _ := strconv.Atoi(countVal)
	return count, nil
}

func (r *redisLimiter) acquire() (net.Conn, error) {
	select {
	case c := <-r.pool:
		if c != nil {
			return c, nil
		}
	default:
	}
	return net.DialTimeout("tcp", r.addr, 2*time.Second)
}

func (r *redisLimiter) release(c net.Conn) {
	select {
	case r.pool <- c:
	default:
		c.Close()
	}
}

// readRESP reads a single RESP value and returns its string representation.
// Only handles integers (:) and simple strings (+) needed for this use case.
func readRESP(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return "", fmt.Errorf("empty RESP line")
	}
	switch line[0] {
	case ':': // integer
		return line[1:], nil
	case '+': // simple string
		return line[1:], nil
	case '-': // error
		return "", fmt.Errorf("redis error: %s", line[1:])
	default:
		return line[1:], nil
	}
}

// newRateLimiter builds a RateLimiter: Redis-backed if redisURL is set,
// otherwise in-memory (single-instance only — safe for one gateway replica).
func newRateLimiter(redisURL string, limit int, window time.Duration) RateLimiter {
	if redisURL != "" {
		rl, err := newRedisLimiter(redisURL, limit, window)
		if err != nil {
			slog.Warn("redis rate limiter unavailable, falling back to memory", "error", err)
		} else {
			slog.Info("rate limiter: redis sliding window", "addr", rl.addr, "limit", limit, "window", window)
			return rl
		}
	}
	slog.Info("rate limiter: in-memory (single instance only)", "limit", limit, "window", window)
	return newMemoryLimiter(limit, window)
}
