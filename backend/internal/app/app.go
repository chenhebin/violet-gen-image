package app

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/ai"
	"yingyan.local/backend/internal/aiconfig"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/audit"
	"yingyan.local/backend/internal/auth"
	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/generation"
	"yingyan.local/backend/internal/handlers"
	"yingyan.local/backend/internal/httpapi"
	"yingyan.local/backend/internal/manage"
	"yingyan.local/backend/internal/platform/database"
	"yingyan.local/backend/internal/prompt"
	"yingyan.local/backend/internal/redemption"
	"yingyan.local/backend/internal/retouch"
	"yingyan.local/backend/internal/storage"
	"yingyan.local/backend/internal/user"
)

type App struct {
	Config config.Config
	Logger *slog.Logger
	DB     *gorm.DB
	Store  storage.Store
	Router *gin.Engine
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := database.Open(cfg.Database)
	if err != nil {
		return nil, err
	}
	store, err := storage.New(cfg.Storage)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("configure object storage: %w", err)
	}
	authService := auth.NewService(db, cfg)
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
	promptService := prompt.New(db, assetService, aiFactory, logger)
	generationService := generation.New(db, creditService, assetService, promptService)
	retouchService := retouch.New(db, creditService, assetService, generationService)
	redemptionService := redemption.New(
		db, creditService, cfg.Security.EncryptionKey, cfg.Security.RedemptionPepper,
	)
	aiConfigService := aiconfig.New(
		db, aiFactory, cfg.Security.EncryptionKey, cfg.Provider.AllowHTTP,
	)
	userService := user.New(db, creditService)
	manageService := manage.New(
		db, creditService, redemptionService, assetService,
		generationService, retouchService, cfg.Security.BcryptCost,
	)
	auditService := audit.New(db, cfg.Security.TokenPepper)
	userHandler := handlers.NewUserHandler(
		userService, redemptionService, assetService, promptService,
		generationService, retouchService,
	)
	adminHandler := handlers.NewAdminHandler(
		manageService, redemptionService, aiConfigService,
		retouchService, assetService, auditService,
	)
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: cfg, Logger: logger, DB: db, Store: store, AuthService: authService,
		UserHandler: userHandler, AdminHandler: adminHandler,
	})
	return &App{
		Config: cfg, Logger: logger, DB: db, Store: store,
		Router: router,
	}, nil
}

func (a *App) Close() error {
	return database.Close(a.DB)
}
