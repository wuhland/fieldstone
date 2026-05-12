package main

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func newProxy(targetURL string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		slog.Error("invalid proxy target URL", "url", targetURL, "error", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad gateway configuration", http.StatusInternalServerError)
		})
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("proxy error", "error", err, "target", targetURL)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"bad gateway"}`))
	}
	return proxy
}
