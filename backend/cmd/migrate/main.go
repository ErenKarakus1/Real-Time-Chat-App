package main

import (
	"context"
	"log"
	"os"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/config"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/db"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/migrations"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	dbPool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer dbPool.Close()

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	runner := migrations.NewRunner(dbPool, migrationsDir)
	if err := runner.Run(ctx); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	log.Println("migrations applied")
}
