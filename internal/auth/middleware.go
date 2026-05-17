package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type MiddlewareConfig struct {
	IssuerURL string
	Audience  string
	Cache     *JWKSCache

	// ResidentIssuerURL, when set, enables a second OIDC issuer for residents
	// (e.g. Login.gov). Tokens from this issuer receive the synthetic role
	// "resident" regardless of what the token claims contain.
	ResidentIssuerURL string
	ResidentCache     *JWKSCache
}

func Middleware(cfg *MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := extractBearer(r)
			if !ok {
				writeUnauthorized(w)
				return
			}

			var claims *Claims
			var err error

			iss := peekIssuer(tokenStr)
			if cfg.ResidentIssuerURL != "" && iss == cfg.ResidentIssuerURL {
				claims, err = parseToken(r.Context(), tokenStr, cfg.ResidentCache, cfg.ResidentIssuerURL, cfg.Audience)
				if err != nil {
					writeUnauthorized(w)
					return
				}
				claims.Roles = []string{"resident"}
			} else {
				claims, err = parseToken(r.Context(), tokenStr, cfg.Cache, cfg.IssuerURL, cfg.Audience)
				if err != nil {
					writeUnauthorized(w)
					return
				}
			}

			// Inject trusted internal headers so downstream services can identify
			// the caller without re-parsing the JWT.
			r.Header.Set("X-Fieldstone-Sub", claims.Subject)
			r.Header.Set("X-Fieldstone-Email", claims.Email)
			if len(claims.Roles) > 0 {
				r.Header.Set("X-Fieldstone-Role", claims.Roles[0])
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

// peekIssuer extracts the iss claim without verifying the token signature.
// Used only to select which key set to validate against; the full validation
// still happens in parseToken.
func peekIssuer(tokenStr string) string {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var c struct {
		Issuer string `json:"iss"`
	}
	json.Unmarshal(payload, &c) //nolint:errcheck — invalid JSON produces empty string, handled by caller
	return c.Issuer
}

func parseToken(ctx context.Context, tokenStr string, cache *JWKSCache, issuerURL, audience string) (*Claims, error) {
	token, err := jwt.Parse(tokenStr,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			kid, _ := t.Header["kid"].(string)
			return cache.GetKey(ctx, kid)
		},
		jwt.WithIssuer(issuerURL),
		jwt.WithAudience(audience),
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
		Issuer:   issuerURL,
		Audience: []string{audience},
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
