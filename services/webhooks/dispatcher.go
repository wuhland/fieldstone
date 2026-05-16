package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fieldstone/fieldstone/internal/events"
	webhooksdb "github.com/fieldstone/fieldstone/services/webhooks/db/generated"
)

const (
	maxConsecutiveFailures = 10
	deliveryTimeout        = 10 * time.Second
)

type deliveryOutcome struct {
	statusCode *int32
	durationMs *int32
	errMsg     *string
	succeeded  bool
}

// deliverToEndpoint dispatches env to a single endpoint, records the attempt in
// the deliveries table, and updates the endpoint's fail_count. If the endpoint
// exceeds maxConsecutiveFailures it is disabled automatically.
func deliverToEndpoint(ctx context.Context, q *webhooksdb.Queries, ep *webhooksdb.Endpoint, env events.EventEnvelope) {
	outcome := attemptDelivery(ep.URL, ep.SecretHash, env)

	if _, err := q.InsertDelivery(ctx, webhooksdb.InsertDeliveryParams{
		EndpointID: ep.ID,
		EventID:    env.ID,
		EventType:  env.EventType,
		StatusCode: outcome.statusCode,
		DurationMs: outcome.durationMs,
		Error:      outcome.errMsg,
	}); err != nil {
		slog.Error("record delivery", "endpoint_id", webhooksdb.UUIDToStr(ep.ID), "error", err)
	}

	if outcome.succeeded {
		if err := q.ResetFailCount(ctx, ep.ID); err != nil {
			slog.Error("reset fail count", "endpoint_id", webhooksdb.UUIDToStr(ep.ID), "error", err)
		}
		return
	}

	newCount, err := q.IncrementFailCount(ctx, ep.ID)
	if err != nil {
		slog.Error("increment fail count", "endpoint_id", webhooksdb.UUIDToStr(ep.ID), "error", err)
		return
	}

	if newCount >= maxConsecutiveFailures {
		slog.Warn("disabling webhook endpoint after consecutive failures",
			"endpoint_id", webhooksdb.UUIDToStr(ep.ID),
			"url", ep.URL,
			"fail_count", newCount,
		)
		if err := q.DisableEndpoint(ctx, ep.ID); err != nil {
			slog.Error("disable endpoint", "endpoint_id", webhooksdb.UUIDToStr(ep.ID), "error", err)
		}
	}
}

// attemptDelivery sends the event to url with exponential backoff retries.
func attemptDelivery(url, secret string, env events.EventEnvelope) deliveryOutcome {
	body, err := json.Marshal(env)
	if err != nil {
		msg := fmt.Sprintf("marshal event: %v", err)
		return deliveryOutcome{errMsg: &msg}
	}

	sig := computeSignature(secret, body)
	delays := []time.Duration{1, 2, 4, 8, 16}

	var outcome deliveryOutcome
	for attempt, delay := range delays {
		start := time.Now()
		sc, sendErr := sendHTTP(url, body, sig, env.EventType)
		elapsed := int32(time.Since(start).Milliseconds())
		outcome.durationMs = &elapsed

		if sendErr != nil {
			msg := sendErr.Error()
			outcome.errMsg = &msg
			slog.Warn("webhook delivery error",
				"attempt", attempt+1, "url", url, "error", sendErr)
		} else {
			outcome.statusCode = &sc
			if sc >= 200 && sc < 300 {
				outcome.succeeded = true
				outcome.errMsg = nil
				return outcome
			}
			msg := fmt.Sprintf("non-2xx status: %d", sc)
			outcome.errMsg = &msg
			slog.Warn("webhook non-2xx",
				"attempt", attempt+1, "url", url, "status", sc)
		}

		if attempt < len(delays)-1 {
			time.Sleep(delay * time.Second)
		}
	}
	return outcome
}

func sendHTTP(url string, body []byte, sig, eventType string) (int32, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Fieldstone-Signature", "sha256="+sig)
	req.Header.Set("X-Fieldstone-Event", eventType)

	client := &http.Client{Timeout: deliveryTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return int32(resp.StatusCode), nil
}

func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
