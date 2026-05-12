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

	"github.com/fieldstone/fieldstone/internal/db"
	"github.com/fieldstone/fieldstone/internal/events"
	"github.com/fieldstone/fieldstone/internal/middleware"
	natsconn "github.com/fieldstone/fieldstone/internal/nats"
	"github.com/fieldstone/fieldstone/services/webhooks/handlers"
	"github.com/go-chi/chi/v5"
)

const version = "0.1.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseDSN)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	nc, js, err := natsconn.Connect(cfg.NATSURL)
	if err != nil {
		slog.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer nc.Drain()

	// TODO(fieldstone): load registered webhook endpoints from DB and set up dispatching
	if err := subscribeToEvents(js, func(env events.EventEnvelope) {
		slog.Info("webhook event received", "event_type", env.EventType)
		// TODO(fieldstone): look up matching registered webhooks and call dispatch()
	}); err != nil {
		slog.Error("failed to subscribe to events", "error", err)
		os.Exit(1)
	}

	h := handlers.NewWebhookHandler()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "webhooks",
			"version": version,
		})
	})

	r.Get("/v1/webhooks", h.ListWebhooks)
	r.Post("/v1/webhooks", h.CreateWebhook)
	r.Get("/v1/webhooks/{id}", h.GetWebhook)
	r.Delete("/v1/webhooks/{id}", h.DeleteWebhook)
	r.Post("/v1/webhooks/{id}/test", h.TestWebhook)

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("service started", "service", "webhooks", "version", version, "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
