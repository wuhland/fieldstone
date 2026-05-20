package main

import (
	"github.com/fieldstone/fieldstone/internal/config"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	config.BaseConfig
	DatabaseDSN  string `env:"WEBHOOKS_DATABASE_DSN" env-required:"true"`
	Addr         string `env:"WEBHOOKS_ADDR"         env-default:":8086"`
	TemporalHost string `env:"TEMPORAL_HOST"         env-default:"localhost:7233"`
}

func loadConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
