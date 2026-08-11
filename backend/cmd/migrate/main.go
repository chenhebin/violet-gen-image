package main

import (
	"context"
	"log/slog"
	"os"

	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/migration"
	"yingyan.local/backend/internal/platform/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load_config_failed", "error", err)
		os.Exit(1)
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		logger.Error("database_connect_failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = database.Close(db) }()

	if err := migration.Apply(context.Background(), db, logger); err != nil {
		logger.Error("migration_failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations_complete")
}
