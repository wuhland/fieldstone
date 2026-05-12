package main

import (
	"github.com/fieldstone/fieldstone/internal/config"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	config.BaseConfig
	DatabaseDSN        string `env:"PERMITS_DATABASE_DSN" env-required:"true"`
	Addr               string `env:"PERMITS_ADDR"         env-default:":8081"`
	WorkflowServiceURL string `env:"WORKFLOW_SERVICE_URL" env-required:"true"`
}

func loadConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
