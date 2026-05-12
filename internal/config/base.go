package config

// BaseConfig is embedded by every service's config struct.
type BaseConfig struct {
	NATSURL     string `env:"NATS_URL"    env-default:"nats://localhost:4222"`
	LogLevel    string `env:"LOG_LEVEL"   env-default:"info"`
	Environment string `env:"ENVIRONMENT" env-default:"production"`
}
