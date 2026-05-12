package main

import (
	"github.com/fieldstone/fieldstone/internal/config"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	config.BaseConfig
	Addr               string `env:"GATEWAY_PORT"           env-default:":8080"`
	OIDCIssuerURL      string `env:"OIDC_ISSUER_URL"        env-required:"true"`
	OIDCAudience       string `env:"OIDC_AUDIENCE"          env-required:"true"`
	AllowedOrigins     string `env:"ALLOWED_ORIGINS"        env-default:"*"`
	PermitsServiceURL  string `env:"PERMITS_SERVICE_URL"    env-required:"true"`
	RequestsServiceURL string `env:"REQUESTS_SERVICE_URL"   env-required:"true"`
	RecordsServiceURL  string `env:"RECORDS_SERVICE_URL"    env-required:"true"`
	IdentityServiceURL string `env:"IDENTITY_SERVICE_URL"   env-required:"true"`
	WorkflowServiceURL string `env:"WORKFLOW_SERVICE_URL"   env-required:"true"`
	WebhooksServiceURL string `env:"WEBHOOKS_SERVICE_URL"   env-required:"true"`
	AuditServiceURL    string `env:"AUDIT_SERVICE_URL"      env-required:"true"`
}

func loadConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
