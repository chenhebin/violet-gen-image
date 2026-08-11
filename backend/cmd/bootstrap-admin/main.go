package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"yingyan.local/backend/internal/bootstrap"
	"yingyan.local/backend/internal/config"
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

	admin, err := bootstrap.CreateFirstAdmin(context.Background(), db, bootstrap.AdminInput{
		Email:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		Password: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		Name:     env("BOOTSTRAP_ADMIN_NAME", "平台管理员"),
	}, cfg.Security.BcryptCost)
	if err != nil {
		if errors.Is(err, bootstrap.ErrAdminAlreadyExists) {
			logger.Error("bootstrap_admin_refused", "error", "an administrator already exists")
		} else {
			logger.Error("bootstrap_admin_failed", "error", err)
		}
		os.Exit(1)
	}

	logger.Info("bootstrap_admin_created",
		"admin_id", admin.ID,
		"email", admin.Email,
		"role", admin.Role,
	)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
