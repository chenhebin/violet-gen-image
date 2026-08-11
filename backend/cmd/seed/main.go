package main

import (
	"context"
	"log/slog"
	"os"

	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/platform/database"
	"yingyan.local/backend/internal/seed"
	"yingyan.local/backend/internal/storage"
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

	store, err := storage.New(cfg.Storage)
	if err != nil {
		logger.Error("storage_config_failed", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()
	if err := store.EnsureBucket(ctx); err != nil {
		logger.Error("storage_prepare_failed", "error", err)
		os.Exit(1)
	}

	if err := seed.Run(ctx, db, store, cfg, logger); err != nil {
		logger.Error("demo_seed_failed", "error", err)
		os.Exit(1)
	}
}
