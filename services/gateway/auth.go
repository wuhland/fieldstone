package main

import (
	"context"
	"log/slog"

	"github.com/fieldstone/fieldstone/internal/auth"
)

func newJWKSCache(ctx context.Context, issuerURL string) (*auth.JWKSCache, error) {
	slog.Info("fetching OIDC discovery", "issuer", issuerURL)
	return auth.NewJWKSCache(ctx, issuerURL)
}
