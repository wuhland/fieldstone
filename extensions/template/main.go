// Package main is a template for Fieldstone extension services.
// Copy this directory and modify it to subscribe to the events your extension needs.
//
// To add to your city's Docker Compose:
//
//	services:
//	  my-extension:
//	    build: ./extensions/my-extension
//	    environment:
//	      NATS_URL: nats://nats:4222
//	      EXTENSION_ADDR: :9000
//	    depends_on:
//	      nats:
//	        condition: service_healthy
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	nats "github.com/nats-io/nats.go"
)

// EventEnvelope matches the structure published by all Fieldstone services.
// Copy this struct into your extension — do not import from the core module.
type EventEnvelope struct {
	ID            string          `json:"id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	SourceService string          `json:"source_service"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	SchemaVersion int             `json:"schema_version"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	addr := getEnv("EXTENSION_ADDR", ":9000")

	nc, err := nats.Connect(natsURL)
	if err != nil {
		slog.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer nc.Drain()

	js, err := nc.JetStream()
	if err != nil {
		slog.Error("failed to create JetStream context", "error", err)
		os.Exit(1)
	}

	// Subscribe to permit created events.
	// Change the subject to match the events you care about.
	// Full subject catalog: see docs/extensions.md
	if _, err := js.Subscribe("fieldstone.permits.permit.created",
		handlePermitCreated,
		nats.Durable("my-extension-permits"),
		nats.ManualAck(),
	); err != nil {
		slog.Error("failed to subscribe", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{Addr: addr, Handler: mux}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("extension started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
		}
	}()

	<-sigCh
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func handlePermitCreated(msg *nats.Msg) {
	var env EventEnvelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		slog.Error("failed to decode event", "error", err)
		msg.Nak()
		return
	}

	// Your extension logic here.
	slog.Info("permit created",
		"event_id", env.ID,
		"occurred_at", env.OccurredAt,
	)

	msg.Ack()
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
