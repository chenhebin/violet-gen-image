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

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/ai"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/generation"
	"yingyan.local/backend/internal/health"
	appmiddleware "yingyan.local/backend/internal/middleware"
	"yingyan.local/backend/internal/platform/database"
	"yingyan.local/backend/internal/retouch"
	"yingyan.local/backend/internal/storage"
	"yingyan.local/backend/internal/worker"
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
		logger.Error("object_storage_config_failed", "error", err)
		os.Exit(1)
	}
	creditService := credit.New(db)
	assetService := asset.New(db, store)
	aiFactory := ai.NewFactory(ai.FactoryConfig{
		EncryptionKey:         cfg.Security.EncryptionKey,
		AllowHTTP:             cfg.Provider.AllowHTTP,
		AllowPrivateNetwork:   cfg.Provider.AllowPrivateNetwork,
		ConnectTimeout:        cfg.Provider.ConnectTimeout,
		RequestTimeout:        cfg.Provider.RequestTimeout,
		ResponseHeaderTimeout: cfg.Provider.ResponseHeaderTimeout,
	})
	processor := generation.NewProcessor(
		db,
		creditService,
		assetService,
		aiFactory,
		cfg.Worker.StaleAfter,
		cfg.Worker.QueueTimeout,
		cfg.Worker.OutputTimeout,
		logger,
	)
	retouchService := retouch.New(db, creditService, assetService, nil)

	checker := health.NewChecker(db, store)
	healthRouter := gin.New()
	healthRouter.Use(appmiddleware.RequestID(), appmiddleware.Recovery(logger), appmiddleware.Logger(logger))
	healthRouter.GET("/health/live", checker.Live)
	healthRouter.GET("/health/ready", checker.Ready)
	healthServer := &http.Server{
		Addr: cfg.Worker.HealthAddr, Handler: healthRouter, ReadHeaderTimeout: 5 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- healthServer.ListenAndServe()
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	runnerErrors := make(chan error, 1)
	go func() {
		runnerErrors <- worker.New(
			logger, cfg.Worker.PollInterval, cfg.Worker.WorkerID, processor,
		).Run(ctx)
	}()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				expired, expireErr := retouchService.ExpireQuotes(ctx, 100)
				if expireErr != nil {
					logger.Error("retouch_quote_expiry_failed", "error", expireErr)
				} else if expired > 0 {
					logger.Info("retouch_quote_expired", "count", expired)
				}
				cleaned, cleanupErr := assetService.CleanupExpired(ctx, 20)
				if cleanupErr != nil {
					logger.Error("asset_cleanup_failed", "error", cleanupErr)
				} else if cleaned > 0 {
					logger.Info("asset_cleanup_completed", "count", cleaned)
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("worker_health_server_failed", "error", err)
		}
		cancel()
	case err := <-runnerErrors:
		if err != nil {
			logger.Error("worker_failed", "error", err)
		}
		cancel()
	}

	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := healthServer.Shutdown(shutdownContext); err != nil {
		logger.Error("worker_health_shutdown_failed", "error", err)
	}
}
