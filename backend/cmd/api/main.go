package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yingyan.local/backend/internal/app"
	"yingyan.local/backend/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load_config_failed", "error", err)
		os.Exit(1)
	}
	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("bootstrap_failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = application.Close() }()

	server := &http.Server{
		Addr:              cfg.App.HTTPAddr,
		Handler:           application.Router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       75 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api_started", "address", cfg.App.HTTPAddr, "environment", cfg.App.Env)
		serverErrors <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case received := <-signals:
		logger.Info("shutdown_signal_received", "signal", received.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api_server_failed", "error", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("api_shutdown_failed", "error", err)
	}
}
