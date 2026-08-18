package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Fatalf("AppEnv = %q, want development", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 10s", cfg.ShutdownTimeout)
	}
	if cfg.BrokerDriver != "memory" {
		t.Fatalf("BrokerDriver = %q, want memory", cfg.BrokerDriver)
	}
}

func TestLoadCustomValues(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_ENV", "TEST")
	t.Setenv("HTTP_ADDR", " 127.0.0.1:9090 ")
	t.Setenv("SHUTDOWN_TIMEOUT", "250ms")
	t.Setenv("BROKER_DRIVER", "MEMORY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppEnv != "test" || cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("unexpected normalized values: %+v", cfg)
	}
	if cfg.ShutdownTimeout != 250*time.Millisecond {
		t.Fatalf("ShutdownTimeout = %s, want 250ms", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "environment", key: "APP_ENV", value: "staging", want: "APP_ENV"},
		{name: "address", key: "HTTP_ADDR", value: " ", want: "HTTP_ADDR"},
		{name: "timeout", key: "SHUTDOWN_TIMEOUT", value: "soon", want: "SHUTDOWN_TIMEOUT"},
		{name: "broker", key: "BROKER_DRIVER", value: "redis", want: "BROKER_DRIVER"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv(tt.key, tt.value)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want an error containing %q", err, tt.want)
			}
		})
	}
}

func TestLoadReportsMultipleInvalidValues(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_ENV", "staging")
	t.Setenv("HTTP_ADDR", " ")
	t.Setenv("SHUTDOWN_TIMEOUT", "soon")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want validation errors")
	}
	for _, expected := range []string{"APP_ENV", "HTTP_ADDR", "SHUTDOWN_TIMEOUT"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Load() error = %q, missing %q", err, expected)
		}
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"APP_ENV", "HTTP_ADDR", "SHUTDOWN_TIMEOUT", "BROKER_DRIVER"} {
		oldValue, existed := os.LookupEnv(key)
		keyCopy, oldValueCopy := key, oldValue
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(keyCopy, oldValueCopy)
				return
			}
			_ = os.Unsetenv(keyCopy)
		})
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
}
