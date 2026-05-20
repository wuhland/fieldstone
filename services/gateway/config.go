package main

import (
	"github.com/fieldstone/fieldstone/internal/config"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	config.BaseConfig
	Addr                  string `env:"GATEWAY_PORT"           env-default:":8080"`
	OIDCIssuerURL         string `env:"OIDC_ISSUER_URL"`
	OIDCAudience          string `env:"OIDC_AUDIENCE"`
	ResidentOIDCIssuerURL string `env:"RESIDENT_OIDC_ISSUER_URL"`
	DevDisableAuth        bool   `env:"DEV_DISABLE_AUTH"       env-default:"false"`
	AllowedOrigins        string `env:"ALLOWED_ORIGINS"        env-default:"*"`
	ServicesConfigPath    string `env:"SERVICES_CONFIG"        env-default:"/etc/fieldstone/services.yaml"`
	RedisURL              string `env:"REDIS_URL"              env-default:""`
	RateLimitPerMin       int    `env:"RATE_LIMIT_PER_MIN"     env-default:"100"`
}

func loadConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
