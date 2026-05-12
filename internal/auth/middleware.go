package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type MiddlewareConfig struct {
	IssuerURL string
	Audience  string
	Cache     *JWKSCache
}

func Middleware(cfg *MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := extractBearer(r)
			if !ok {
				writeUnauthorized(w)
				return
			}

			claims, err := parseToken(r.Context(), tokenStr, cfg)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithClaims(r.Context(), claims)))
		})
	}
}

func extractBearer(r *http.Request) (string, bool) {
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(hdr, "Bearer "), true
}

func parseToken(ctx context.Context, tokenStr string, cfg *MiddlewareConfig) (*Claims, error) {
	token, err := jwt.Parse(tokenStr,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			kid, _ := t.Header["kid"].(string)
			return cfg.Cache.GetKey(ctx, kid)
		},
		jwt.WithIssuer(cfg.IssuerURL),
		jwt.WithAudience(cfg.Audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}

	mc, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	c := &Claims{
		Issuer:   cfg.IssuerURL,
		Audience: []string{cfg.Audience},
	}
	if sub, ok := mc["sub"].(string); ok {
		c.Subject = sub
	}
	if email, ok := mc["email"].(string); ok {
		c.Email = email
	}
	if rolesRaw, ok := mc["roles"].([]interface{}); ok {
		for _, r := range rolesRaw {
			if s, ok := r.(string); ok {
				c.Roles = append(c.Roles, s)
			}
		}
	}
	return c, nil
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
