package main

import (
	"github.com/fieldstone/fieldstone/internal/config"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	config.BaseConfig
	Addr         string `env:"WORKFLOW_WORKER_ADDR"   env-default:":8085"`
	TemporalHost string `env:"TEMPORAL_HOST"          env-required:"true"`
	WorkflowsDir string `env:"WORKFLOWS_DIR"          env-default:"/etc/fieldstone/workflows"`
	// DB DSNs for activities (resident/timer-driven transitions).
	// These are the same DSNs used by each domain service.
	PermitsDSN  string `env:"PERMITS_DATABASE_DSN"`
	RequestsDSN string `env:"REQUESTS_DATABASE_DSN"`
	RecordsDSN  string `env:"RECORDS_DATABASE_DSN"`
}

func loadConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
