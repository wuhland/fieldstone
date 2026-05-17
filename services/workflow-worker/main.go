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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.temporal.io/sdk/worker"

	internaldb "github.com/fieldstone/fieldstone/internal/db"
	"github.com/fieldstone/fieldstone/internal/middleware"
	temporalclient "github.com/fieldstone/fieldstone/internal/temporal"
	"github.com/fieldstone/fieldstone/internal/workflows"
	workeractivities "github.com/fieldstone/fieldstone/services/workflow-worker/activities"
	workerworkflows "github.com/fieldstone/fieldstone/services/workflow-worker/workflows"
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

	wfConfigs, err := workflows.LoadWorkflows(cfg.WorkflowsDir)
	if err != nil {
		slog.Error("failed to load workflows", "error", err, "dir", cfg.WorkflowsDir)
		os.Exit(1)
	}
	slog.Info("workflows loaded", "count", len(wfConfigs))

	tc, err := temporalclient.NewClient(cfg.TemporalHost)
	if err != nil {
		slog.Error("failed to connect to Temporal", "error", err)
		os.Exit(1)
	}
	defer tc.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to domain DBs for activities (optional; only needed when signal/timer
	// activities fire). If DSNs are empty the activity structs have nil pools and
	// activities will return an error if called — acceptable until those paths are wired.
	var permitActs *workeractivities.PermitActivities
	var requestActs *workeractivities.RequestActivities
	var recordsActs *workeractivities.RecordsActivities

	if cfg.PermitsDSN != "" {
		pool, err := internaldb.Connect(ctx, cfg.PermitsDSN)
		if err != nil {
			slog.Error("failed to connect to permits DB", "error", err)
			os.Exit(1)
		}
		defer pool.Close()
		permitActs = workeractivities.NewPermitActivities(pool)
	}
	if cfg.RequestsDSN != "" {
		pool, err := internaldb.Connect(ctx, cfg.RequestsDSN)
		if err != nil {
			slog.Error("failed to connect to requests DB", "error", err)
			os.Exit(1)
		}
		defer pool.Close()
		requestActs = workeractivities.NewRequestActivities(pool)
	}
	if cfg.RecordsDSN != "" {
		pool, err := internaldb.Connect(ctx, cfg.RecordsDSN)
		if err != nil {
			slog.Error("failed to connect to records DB", "error", err)
			os.Exit(1)
		}
		defer pool.Close()
		recordsActs = workeractivities.NewRecordsActivities(pool)
	}

	w := worker.New(tc, temporalclient.TaskQueue, worker.Options{})
	w.RegisterWorkflow(workerworkflows.PermitWorkflow)
	w.RegisterWorkflow(workerworkflows.ServiceRequestWorkflow)
	w.RegisterWorkflow(workerworkflows.FOIARequestWorkflow)
	if permitActs != nil {
		w.RegisterActivity(permitActs)
	}
	if requestActs != nil {
		w.RegisterActivity(requestActs)
	}
	if recordsActs != nil {
		w.RegisterActivity(recordsActs)
	}

	// HTTP server: serves /v1/workflow/* (same endpoints as the retired workflow
	// service) so domain services can query valid statuses/transitions and initial
	// status at startup. The gateway /v1/workflow mount points here instead.
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.Metrics("workflow-worker"))
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "workflow-worker", "version": version}) //nolint:errcheck
	})

	wh := newWorkflowHandler(wfConfigs)
	r.Route("/v1/workflow/{resource_type}", func(r chi.Router) {
		r.Get("/statuses", wh.GetStatuses)
		r.Get("/transitions", wh.GetTransitions)
		r.Post("/validate", wh.ValidateTransition)
		r.Get("/initial", wh.GetInitialStatus)
		r.Get("/config", wh.GetConfig) // full WorkflowConfig for workflow start
	})

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("service started", "service", "workflow-worker", "version", version, "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
		}
	}()

	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			slog.Error("temporal worker error", "error", err)
		}
	}()

	<-sigCh
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
