package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fieldstone/fieldstone/internal/auth"
	"github.com/fieldstone/fieldstone/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// staffAuthMW validates staff OIDC JWTs (single issuer).
	// residentAuthMW validates staff OR resident JWTs (multi-issuer when configured).
	// In dev mode both are passthroughs so the services can be exercised without
	// an OIDC provider. Never set DEV_DISABLE_AUTH in production.
	var staffAuthMW, residentAuthMW func(http.Handler) http.Handler

	if cfg.DevDisableAuth {
		slog.Warn("DEV_DISABLE_AUTH=true — authentication is disabled, do not use in production")
		staffAuthMW = func(next http.Handler) http.Handler { return next }
		residentAuthMW = func(next http.Handler) http.Handler { return next }
	} else {
		if cfg.OIDCIssuerURL == "" || cfg.OIDCAudience == "" {
			slog.Error("OIDC_ISSUER_URL and OIDC_AUDIENCE are required when DEV_DISABLE_AUTH is false")
			os.Exit(1)
		}

		staffCache, err := newJWKSCache(ctx, cfg.OIDCIssuerURL)
		if err != nil {
			slog.Error("failed to initialize staff JWKS cache", "error", err)
			os.Exit(1)
		}

		staffAuthMW = auth.Middleware(&auth.MiddlewareConfig{
			IssuerURL: cfg.OIDCIssuerURL,
			Audience:  cfg.OIDCAudience,
			Cache:     staffCache,
		})

		residentCfg := &auth.MiddlewareConfig{
			IssuerURL: cfg.OIDCIssuerURL,
			Audience:  cfg.OIDCAudience,
			Cache:     staffCache,
		}
		if cfg.ResidentOIDCIssuerURL != "" {
			residentCache, err := newJWKSCache(ctx, cfg.ResidentOIDCIssuerURL)
			if err != nil {
				slog.Error("failed to initialize resident JWKS cache", "error", err)
				os.Exit(1)
			}
			residentCfg.ResidentIssuerURL = cfg.ResidentOIDCIssuerURL
			residentCfg.ResidentCache = residentCache
			slog.Info("resident OIDC issuer configured", "issuer", cfg.ResidentOIDCIssuerURL)
		}
		residentAuthMW = auth.Middleware(residentCfg)
	}

	rl := newRateLimiter(cfg.RedisURL, cfg.RateLimitPerMin, time.Minute)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.Metrics("gateway"))
	// Strip any X-Fieldstone-* headers from incoming client requests to prevent
	// spoofing of gateway-injected identity headers.
	r.Use(stripInternalHeaders)

	registerDocRoutes(r)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "gateway",
			"version": version,
		})
	})

	// Public route: permit status is readable without authentication.
	r.Group(func(r chi.Router) {
		r.Use(rateLimitMiddleware(rl, cfg.RateLimitPerMin))
		r.Get("/v1/permits/{id}/status", newProxy(cfg.PermitsServiceURL).ServeHTTP)
	})

	// Resident-capable routes: accept both staff and resident JWTs.
	// Domain services use the X-Fieldstone-Role header to apply row-level access
	// control (residents can only read their own submissions).
	r.Group(func(r chi.Router) {
		r.Use(rateLimitMiddleware(rl, cfg.RateLimitPerMin))
		r.Use(residentAuthMW)
		r.Post("/v1/requests", newProxy(cfg.RequestsServiceURL).ServeHTTP)
		r.Get("/v1/requests/{id}", newProxy(cfg.RequestsServiceURL).ServeHTTP)
		r.Post("/v1/records/foia", newProxy(cfg.RecordsServiceURL).ServeHTTP)
		r.Get("/v1/records/foia/{id}", newProxy(cfg.RecordsServiceURL).ServeHTTP)
		r.Post("/v1/permits", newProxy(cfg.PermitsServiceURL).ServeHTTP)
		r.Get("/v1/permits/{id}", newProxy(cfg.PermitsServiceURL).ServeHTTP)
	})

	// Staff-only routes: reject resident JWTs (wrong issuer → 401).
	r.Group(func(r chi.Router) {
		r.Use(staffAuthMW)
		// Requests management
		r.Get("/v1/requests", newProxy(cfg.RequestsServiceURL).ServeHTTP)
		r.Patch("/v1/requests/{id}/status", newProxy(cfg.RequestsServiceURL).ServeHTTP)
		r.Patch("/v1/requests/{id}/assign", newProxy(cfg.RequestsServiceURL).ServeHTTP)
		// Records management
		r.Get("/v1/records/foia", newProxy(cfg.RecordsServiceURL).ServeHTTP)
		r.Patch("/v1/records/foia/{id}/status", newProxy(cfg.RecordsServiceURL).ServeHTTP)
		// Permits management
		r.Get("/v1/permits", newProxy(cfg.PermitsServiceURL).ServeHTTP)
		r.Patch("/v1/permits/{id}/status", newProxy(cfg.PermitsServiceURL).ServeHTTP)
		r.Post("/v1/permits/{id}/inspections", newProxy(cfg.PermitsServiceURL).ServeHTTP)
		r.Patch("/v1/permits/{id}/inspections/{iid}", newProxy(cfg.PermitsServiceURL).ServeHTTP)
		// Identity, config, and platform services
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

// stripInternalHeaders removes X-Fieldstone-* headers from incoming client
// requests. The auth middleware re-sets these after JWT validation, so only
// gateway-validated values reach downstream services.
func stripInternalHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key := range r.Header {
			if strings.HasPrefix(key, "X-Fieldstone-") {
				delete(r.Header, key)
			}
		}
		next.ServeHTTP(w, r)
	})
}
