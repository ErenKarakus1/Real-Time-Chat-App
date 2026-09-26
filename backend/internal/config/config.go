package config

import (
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	GinMode           string
	DatabaseURL       string
	RedisURL          string
	JWTSecret         string
	CORSAllowedOrigin string
}

const defaultCORSAllowedOrigin = "http://localhost:5173,http://127.0.0.1:5173"

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:              getEnv("PORT", "8080"),
		GinMode:           getEnv("GIN_MODE", "debug"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		RedisURL:          getEnv("REDIS_URL", ""),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		CORSAllowedOrigin: getEnv("CORS_ALLOWED_ORIGIN", defaultCORSAllowedOrigin),
	}
}

func (c Config) CORSAllowedOrigins() []string {
	var origins []string
	for _, origin := range strings.Split(c.CORSAllowedOrigin, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins = append(origins, origin)
		}
	}

	return origins
}

func (c Config) ValidateServer() error {
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.RedisURL == "" {
		missing = append(missing, "REDIS_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}

	return requiredError(missing)
}

func (c Config) ValidateMigrate() error {
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	return requiredError(missing)
}

func requiredError(missing []string) error {
	if len(missing) == 0 {
		return nil
	}

	return errors.New("missing required environment variables: " + strings.Join(missing, ", "))
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
