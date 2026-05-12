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
	"github.com/fieldstone/fieldstone/internal/middleware"
	"github.com/fieldstone/fieldstone/services/identity/handlers"
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

	userH := handlers.NewUserHandler()
	deptH := handlers.NewDepartmentHandler()
	schemaH := handlers.NewSchemaHandler()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"status":  "ok",
			"service": "identity",
			"version": version,
		})
	})

	r.Get("/v1/departments", deptH.ListDepartments)
	r.Post("/v1/departments", deptH.CreateDepartment)
	r.Get("/v1/users", userH.ListUsers)
	r.Get("/v1/users/me", userH.GetMe)
	r.Post("/v1/users", userH.CreateUser)
	r.Get("/v1/config/schemas/{resource_type}", schemaH.GetSchema)
	r.Put("/v1/config/schemas/{resource_type}", schemaH.PutSchema)

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("service started", "service", "identity", "version", version, "addr", cfg.Addr)
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
