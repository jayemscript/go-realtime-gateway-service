package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultAppEnv          = "development"
	defaultHTTPAddr        = ":8080"
	defaultShutdownTimeout = 10 * time.Second
	defaultBrokerDriver    = "memory"
)

// Config contains the settings needed to start the Phase 1 process.
// Dependency-specific settings are added in the phases that implement them.
type Config struct {
	AppEnv          string
	HTTPAddr        string
	ShutdownTimeout time.Duration
	BrokerDriver    string
}

// Load reads and validates process configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:          envOrDefault("APP_ENV", defaultAppEnv),
		HTTPAddr:        envOrDefault("HTTP_ADDR", defaultHTTPAddr),
		ShutdownTimeout: defaultShutdownTimeout,
		BrokerDriver:    envOrDefault("BROKER_DRIVER", defaultBrokerDriver),
	}

	var problems []string
	cfg.AppEnv = strings.ToLower(strings.TrimSpace(cfg.AppEnv))
	if cfg.AppEnv != "development" && cfg.AppEnv != "test" && cfg.AppEnv != "production" {
		problems = append(problems, "APP_ENV must be development, test, or production")
	}

	cfg.HTTPAddr = strings.TrimSpace(cfg.HTTPAddr)
	if cfg.HTTPAddr == "" {
		problems = append(problems, "HTTP_ADDR must not be empty")
	}

	cfg.BrokerDriver = strings.ToLower(strings.TrimSpace(cfg.BrokerDriver))
	if cfg.BrokerDriver != "memory" {
		problems = append(problems, "BROKER_DRIVER must be memory in Phase 1; Redis is added in a later phase")
	}

	if rawTimeout, ok := os.LookupEnv("SHUTDOWN_TIMEOUT"); ok {
		parsedTimeout, err := time.ParseDuration(strings.TrimSpace(rawTimeout))
		if err != nil || parsedTimeout <= 0 {
			problems = append(problems, "SHUTDOWN_TIMEOUT must be a positive duration such as 10s")
		} else {
			cfg.ShutdownTimeout = parsedTimeout
		}
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("%s", strings.Join(problems, "; "))
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}
