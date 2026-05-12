package auth

import "context"

type contextKey string

const claimsKey contextKey = "auth_claims"

type Claims struct {
	Subject  string   `json:"sub"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles,omitempty"`
	Issuer   string   `json:"iss"`
	Audience []string `json:"aud"`
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

func ContextWithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}
