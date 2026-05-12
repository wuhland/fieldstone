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

	"github.com/go-chi/chi/v5"
	"github.com/fieldstone/fieldstone/internal/db"
	"github.com/fieldstone/fieldstone/internal/middleware"
	natsconn "github.com/fieldstone/fieldstone/internal/nats"
	"github.com/fieldstone/fieldstone/services/audit/handlers"
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

	if err := subscribeToAll(js); err != nil {
		slog.Error("failed to subscribe to events", "error", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)

	r.Get("/v1/audit", handlers.ListEvents)
	r.Get("/v1/audit/{id}", handlers.GetEvent)

	r.Get("/health", healthHandler("audit"))

	srv := &http.Server{Addr: cfg.Addr, Handler: r, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("service started", "service", "audit", "version", version, "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

func healthHandler(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": name, "version": version})
	}
}
