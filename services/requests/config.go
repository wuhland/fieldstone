package main

import (
	"github.com/fieldstone/fieldstone/internal/config"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	config.BaseConfig
	DatabaseDSN        string `env:"REQUESTS_DATABASE_DSN" env-required:"true"`
	Addr               string `env:"REQUESTS_ADDR"         env-default:":8082"`
	WorkflowServiceURL string `env:"WORKFLOW_SERVICE_URL"  env-required:"true"`
	IdentityServiceURL string `env:"IDENTITY_SERVICE_URL"  env-required:"true"`
}

func loadConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
