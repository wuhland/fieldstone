package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

// ServiceEntry describes a single downstream service in services.yaml.
type ServiceEntry struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

// ServicesConfig is the top-level structure of services.yaml.
type ServicesConfig struct {
	Services map[string]ServiceEntry `yaml:"services"`
}

func loadServicesConfig(path string) (*ServicesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var sc ServicesConfig
	if err := yaml.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parse services config: %w", err)
	}
	if sc.Services == nil {
		sc.Services = make(map[string]ServiceEntry)
	}
	return &sc, nil
}

// validate fails fast if any enabled service is missing a URL.
func (sc *ServicesConfig) validate() error {
	for name, svc := range sc.Services {
		if svc.Enabled && svc.URL == "" {
			return fmt.Errorf("service %q is enabled but has no url", name)
		}
	}
	return nil
}

// handler returns the proxy handler for a named service when it is enabled,
// or a 503 Service Unavailable handler when it is disabled or absent.
// The returned value satisfies both http.Handler and can be used with r.Mount.
func (sc *ServicesConfig) handler(name string) http.Handler {
	svc, ok := sc.Services[name]
	if !ok || !svc.Enabled {
		slog.Info("service disabled — routes will return 503", "service", name)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"error":"service not enabled","service":%q}`, name)
		})
	}
	return newProxy(svc.URL)
}
