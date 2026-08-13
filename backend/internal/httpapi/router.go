package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/auth"
	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/handlers"
	"yingyan.local/backend/internal/health"
	appmiddleware "yingyan.local/backend/internal/middleware"
	"yingyan.local/backend/internal/respond"
	"yingyan.local/backend/internal/storage"
)

type Dependencies struct {
	Config       config.Config
	Logger       *slog.Logger
	DB           *gorm.DB
	Store        storage.Store
	AuthService  *auth.Service
	UserHandler  *handlers.UserHandler
	AdminHandler *handlers.AdminHandler
}

func NewRouter(deps Dependencies) *gin.Engine {
	binding.EnableDecoderDisallowUnknownFields = true
	if deps.Config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 20
	engine.Use(
		appmiddleware.RequestID(),
		appmiddleware.Recovery(deps.Logger),
		appmiddleware.CORS(deps.Config.App.AllowedOrigins),
		appmiddleware.SecurityHeaders(),
		appmiddleware.Logger(deps.Logger),
	)

	checker := health.NewChecker(deps.DB, deps.Store)
	engine.GET("/health/live", checker.Live)
	engine.GET("/health/ready", checker.Ready)

	authHandler := auth.NewHandler(deps.AuthService, deps.Config)
	authLimiter := appmiddleware.NewRateLimiter(20, time.Minute)

	api := engine.Group(deps.Config.App.APIBasePath)
	userAuth := api.Group("/auth")
	userRateLimit := authLimiter.Middleware(func(c *gin.Context) string {
		return "user:" + c.ClientIP()
	})
	userAuth.POST("/register", userRateLimit, authHandler.Register)
	userAuth.POST("/login", userRateLimit, authHandler.LoginUser)
	userAuth.GET("/session", auth.RequireUser(deps.AuthService, deps.Config), authHandler.UserSession)
	userAuth.POST("/logout", auth.RequireUser(deps.AuthService, deps.Config), authHandler.LogoutUser)

	adminAuth := api.Group("/manage/auth")
	adminRateLimit := authLimiter.Middleware(func(c *gin.Context) string {
		return "manage:" + c.ClientIP()
	})
	adminAuth.POST("/login", adminRateLimit, authHandler.LoginAdmin)
	adminAuth.GET("/session", auth.RequireAdmin(deps.AuthService, deps.Config), authHandler.AdminSession)
	adminAuth.POST(
		"/logout",
		auth.RequireAdmin(deps.AuthService, deps.Config),
		auth.RequireAdminCSRF(deps.Config),
		authHandler.LogoutAdmin,
	)

	userAPI := api.Group("")
	userAPI.Use(auth.RequireUser(deps.AuthService, deps.Config))
	userAPI.GET("/me", deps.UserHandler.Me)
	userAPI.GET("/entitlements", deps.UserHandler.Entitlement)
	userAPI.GET("/usage/ledger", deps.UserHandler.Ledger)
	userAPI.POST("/usage/quote", deps.UserHandler.Quote)
	userAPI.POST("/redemptions/claim", deps.UserHandler.ClaimRedemption)
	userAPI.POST("/assets", deps.UserHandler.UploadAsset)
	userAPI.DELETE("/assets/:assetId", deps.UserHandler.DeleteAsset)
	userAPI.POST("/prompts/optimize", deps.UserHandler.OptimizePrompt)
	userAPI.POST("/prompts/confirm", deps.UserHandler.ConfirmPrompt)
	userAPI.POST("/generations", deps.UserHandler.CreateGeneration)
	userAPI.GET("/tasks", deps.UserHandler.ListTasks)
	userAPI.GET("/tasks/:taskId", deps.UserHandler.GetTask)
	userAPI.POST("/tasks/:taskId/cancel", deps.UserHandler.CancelTask)
	userAPI.GET("/retouch-tickets", deps.UserHandler.ListRetouch)
	userAPI.GET("/retouch-tickets/:ticketId", deps.UserHandler.GetRetouch)
	userAPI.POST("/tasks/:taskId/retouch-tickets", deps.UserHandler.CreateRetouch)
	userAPI.POST("/retouch-tickets/:ticketId/quote/accept", deps.UserHandler.AcceptRetouchQuote)
	userAPI.POST("/retouch-tickets/:ticketId/cancel", deps.UserHandler.CancelRetouch)
	userAPI.POST("/retouch-tickets/:ticketId/revisions", deps.UserHandler.ReviseRetouch)
	userAPI.POST("/retouch-tickets/:ticketId/confirm", deps.UserHandler.ConfirmRetouch)

	manageAPI := api.Group("/manage")
	manageAPI.Use(
		auth.RequireAdmin(deps.AuthService, deps.Config),
		auth.RequireAdminCSRF(deps.Config),
		appmiddleware.RequireIdempotencyKey(),
	)
	manageAPI.GET("/dashboard", deps.AdminHandler.Dashboard)
	manageAPI.GET(
		"/audit-logs",
		auth.RequirePermission(auth.PermissionPlatformManage),
		deps.AdminHandler.ListAudits,
	)

	platform := manageAPI.Group("")
	platform.Use(auth.RequirePermission(auth.PermissionPlatformManage))
	platform.GET("/redemption-codes", deps.AdminHandler.ListRedemptionCodes)
	platform.GET("/redemption-codes/:codeId", deps.AdminHandler.GetRedemptionCode)
	platform.GET("/redemption-batches", deps.AdminHandler.ListRedemptionBatches)
	platform.GET("/redemption-batches/:batchId", deps.AdminHandler.GetRedemptionBatch)
	platform.POST("/redemption-batches", deps.AdminHandler.CreateRedemptionBatch)
	platform.PATCH("/redemption-batches/:batchId", deps.AdminHandler.UpdateRedemptionBatch)
	platform.POST("/redemption-codes/:codeId/reveal", deps.AdminHandler.RevealRedemptionCode)
	platform.POST("/redemption-batches/:batchId/reveal", deps.AdminHandler.RevealRedemptionBatch)
	platform.POST("/redemption-batches/:batchId/export", deps.AdminHandler.ExportRedemptionBatch)
	platform.POST("/redemption-codes/disable", deps.AdminHandler.DisableRedemptionCodes)
	platform.POST("/redemption-codes/extend", deps.AdminHandler.ExtendRedemptionCodes)
	platform.GET("/ai-providers", deps.AdminHandler.ListProviders)
	platform.POST("/ai-providers", deps.AdminHandler.CreateProvider)
	platform.PATCH("/ai-providers/:providerId", deps.AdminHandler.UpdateProvider)
	platform.DELETE("/ai-providers/:providerId", deps.AdminHandler.DeleteProvider)
	platform.POST("/ai-providers/:providerId/test", deps.AdminHandler.TestProvider)
	platform.POST("/ai-providers/:providerId/rotate-key", deps.AdminHandler.RotateProviderKey)
	platform.GET("/ai-models", deps.AdminHandler.ListModels)
	platform.POST("/ai-models", deps.AdminHandler.CreateModel)
	platform.PATCH("/ai-models/:modelId", deps.AdminHandler.UpdateModel)
	platform.DELETE("/ai-models/:modelId", deps.AdminHandler.DeleteModel)
	platform.POST("/ai-models/:modelId/test", deps.AdminHandler.TestModel)
	platform.GET("/platform-model-bindings", deps.AdminHandler.GetBindings)
	platform.POST("/platform-model-bindings", deps.AdminHandler.BindModel)
	platform.GET("/users", deps.AdminHandler.ListUsers)
	platform.GET("/users/:userId", deps.AdminHandler.GetUser)
	platform.POST("/users/:userId/status", deps.AdminHandler.SetUserStatus)
	platform.POST("/users/:userId/reset-password", deps.AdminHandler.ResetUserPassword)
	platform.POST("/users/:userId/adjust-credits", deps.AdminHandler.AdjustUserCredits)
	platform.GET("/usage-ledger", deps.AdminHandler.ListUsageLedger)
	platform.GET("/generation-tasks", deps.AdminHandler.ListTasks)
	platform.GET("/generation-tasks/:taskId", deps.AdminHandler.GetTask)
	platform.GET("/assets", deps.AdminHandler.ListAssets)
	platform.GET("/assets/:assetId", deps.AdminHandler.GetAsset)
	platform.POST("/assets/:assetId/signed-url", deps.AdminHandler.SignAsset)
	platform.POST("/assets/:assetId/retain", deps.AdminHandler.RetainAsset)
	platform.POST("/assets/:assetId/cleanup", deps.AdminHandler.CleanupAsset)

	retouchAPI := manageAPI.Group("/retouch-tickets")
	retouchAPI.Use(auth.RequirePermission(auth.PermissionRetouchManage))
	retouchAPI.GET("", deps.AdminHandler.ListRetouchTickets)
	retouchAPI.GET("/:ticketId", deps.AdminHandler.GetRetouchTicket)
	retouchAPI.POST("/:ticketId/quote", deps.AdminHandler.QuoteRetouch)
	retouchAPI.POST("/:ticketId/start", deps.AdminHandler.StartRetouch)
	retouchAPI.POST("/:ticketId/deliver", deps.AdminHandler.DeliverRetouch)
	retouchAPI.POST("/:ticketId/reject", deps.AdminHandler.RejectRetouch)
	retouchAPI.POST("/:ticketId/fail", deps.AdminHandler.FailRetouch)

	engine.NoRoute(func(c *gin.Context) {
		respond.Error(c, apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "接口不存在", nil))
	})
	engine.NoMethod(func(c *gin.Context) {
		respond.Error(c, apierror.New(http.StatusMethodNotAllowed, apierror.CodeInvalidInput, "请求方法不支持", nil))
	})
	return engine
}
