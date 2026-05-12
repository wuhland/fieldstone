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

	"github.com/fieldstone/fieldstone/internal/auth"
	"github.com/fieldstone/fieldstone/internal/middleware"
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

	// Build the auth middleware. In dev (DEV_DISABLE_AUTH=true) all requests
	// are treated as authenticated so the service can be exercised without an
	// OIDC provider. Never set this in production.
	var authMW func(http.Handler) http.Handler
	if cfg.DevDisableAuth {
		slog.Warn("DEV_DISABLE_AUTH=true — authentication is disabled, do not use in production")
		authMW = func(next http.Handler) http.Handler { return next }
	} else {
		if cfg.OIDCIssuerURL == "" || cfg.OIDCAudience == "" {
			slog.Error("OIDC_ISSUER_URL and OIDC_AUDIENCE are required when DEV_DISABLE_AUTH is false")
			os.Exit(1)
		}
		jwksCache, err := newJWKSCache(ctx, cfg.OIDCIssuerURL)
		if err != nil {
			slog.Error("failed to initialize JWKS cache", "error", err)
			os.Exit(1)
		}
		authMW = auth.Middleware(&auth.MiddlewareConfig{
			IssuerURL: cfg.OIDCIssuerURL,
			Audience:  cfg.OIDCAudience,
			Cache:     jwksCache,
		})
	}

	rl := newRateLimiter(100, time.Minute) // 100 req/min per IP on public routes

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)

	// Health — public, no auth
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "gateway",
			"version": version,
		})
	})

	// Public routes (rate-limited, no auth required)
	// POST /v1/requests — public so citizens can submit service requests
	// GET /v1/permits/:id/status — public permit status lookup
	r.Group(func(r chi.Router) {
		r.Use(rl.Middleware)
		r.Post("/v1/requests", newProxy(cfg.RequestsServiceURL).ServeHTTP)
		r.Get("/v1/permits/{id}/status", newProxy(cfg.PermitsServiceURL).ServeHTTP)
		r.Post("/v1/records/foia", newProxy(cfg.RecordsServiceURL).ServeHTTP)
	})

	// Authenticated routes — staff only
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Mount("/v1/permits", newProxy(cfg.PermitsServiceURL))
		r.Mount("/v1/requests", newProxy(cfg.RequestsServiceURL))
		r.Mount("/v1/records", newProxy(cfg.RecordsServiceURL))
		r.Mount("/v1/users", newProxy(cfg.IdentityServiceURL))
		r.Mount("/v1/departments", newProxy(cfg.IdentityServiceURL))
		r.Mount("/v1/config", newProxy(cfg.IdentityServiceURL))
		r.Mount("/v1/audit", newProxy(cfg.AuditServiceURL))
		r.Mount("/v1/webhooks", newProxy(cfg.WebhooksServiceURL))
		r.Mount("/v1/workflow", newProxy(cfg.WorkflowServiceURL))
	})

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("service started", "service", "gateway", "version", version, "addr", cfg.Addr)
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
