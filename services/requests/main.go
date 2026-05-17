package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/fieldstone/fieldstone/internal/db"
	"github.com/fieldstone/fieldstone/internal/middleware"
	natsconn "github.com/fieldstone/fieldstone/internal/nats"
	"github.com/fieldstone/fieldstone/internal/outbox"
	requestsdb "github.com/fieldstone/fieldstone/services/requests/db/generated"
	"github.com/fieldstone/fieldstone/services/requests/handlers"
)

//go:embed db/migrations/*.sql
var migrationFiles embed.FS

const version = "0.1.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseDSN)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool, migrationFiles); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	nc, js, err := natsconn.Connect(cfg.NATSURL)
	if err != nil {
		slog.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer nc.Drain()

	pub := &outbox.Publisher{}
	go outbox.Run(ctx, pool, js)
	wf, err := newWorkflowClient(cfg.TemporalHost, cfg.WorkflowServiceURL)
	if err != nil {
		slog.Error("failed to create workflow client", "error", err)
		os.Exit(1)
	}
	sv := newSchemaValidator(cfg.IdentityServiceURL)

	h := handlers.New(pool, requestsdb.New(pool), pub, wf, sv)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.Metrics("requests"))

	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok", "service": "requests", "version": version,
		})
	})

	r.Get("/v1/requests", h.ListRequests)
	r.Post("/v1/requests", h.CreateRequest)
	r.Get("/v1/requests/{id}", h.GetRequest)
	r.Patch("/v1/requests/{id}/status", h.UpdateRequestStatus)
	r.Patch("/v1/requests/{id}/assign", h.AssignRequest)

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("service started", "service", "requests", "version", version, "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	cancel() // stop outbox poller and other background goroutines
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
