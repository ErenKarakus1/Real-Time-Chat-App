package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	GinMode     string
	DatabaseURL string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:        getEnv("PORT", "8080"),
		GinMode:     getEnv("GIN_MODE", "debug"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
