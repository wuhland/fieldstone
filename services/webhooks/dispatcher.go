package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fieldstone/fieldstone/internal/events"
)

// dispatch sends the event to a webhook URL with HMAC-SHA256 signature and retries.
// This is a stub implementation — full retry/persistence logic is TODO.
func dispatch(url, secret string, env events.EventEnvelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	sig := computeSignature(secret, body)

	var lastErr error
	delays := []time.Duration{1, 2, 4, 8, 16}
	for attempt, delay := range delays {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Fieldstone-Signature", "sha256="+sig)
		req.Header.Set("X-Fieldstone-Event", env.EventType)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			slog.Warn("webhook delivery failed", "attempt", attempt+1, "url", url, "error", err)
			time.Sleep(delay * time.Second)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("non-2xx status: %d", resp.StatusCode)
		slog.Warn("webhook delivery non-2xx", "attempt", attempt+1, "url", url, "status", resp.StatusCode)
		time.Sleep(delay * time.Second)
	}
	return fmt.Errorf("webhook delivery failed after %d attempts: %w", len(delays), lastErr)
}

func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
