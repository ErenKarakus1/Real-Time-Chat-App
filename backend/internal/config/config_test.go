package config

import (
	"strings"
	"testing"
)

func TestValidateServerRequiresRuntimeConfig(t *testing.T) {
	err := (Config{}).ValidateServer()
	if err == nil {
		t.Fatalf("expected validation error")
	}

	message := err.Error()
	for _, required := range []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET"} {
		if !strings.Contains(message, required) {
			t.Fatalf("error %q does not contain %s", message, required)
		}
	}
}

func TestValidateServerAllowsDefaultsForPortAndGinMode(t *testing.T) {
	cfg := Config{
		DatabaseURL: "postgres://example",
		RedisURL:    "redis://example",
		JWTSecret:   "secret",
	}

	if err := cfg.ValidateServer(); err != nil {
		t.Fatalf("validate server: %v", err)
	}
}

func TestValidateMigrateOnlyRequiresDatabaseURL(t *testing.T) {
	cfg := Config{
		DatabaseURL: "postgres://example",
	}

	if err := cfg.ValidateMigrate(); err != nil {
		t.Fatalf("validate migrate: %v", err)
	}

	if err := (Config{}).ValidateMigrate(); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("missing database error = %v, want DATABASE_URL", err)
	}
}
