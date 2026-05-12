package main

import (
	"github.com/fieldstone/fieldstone/internal/config"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	config.BaseConfig
	Addr         string `env:"WORKFLOW_ADDR"  env-default:":8085"`
	WorkflowsDir string `env:"WORKFLOWS_DIR"  env-default:"/etc/fieldstone/workflows"`
}

func loadConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
