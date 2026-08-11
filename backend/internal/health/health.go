package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/migration"
	"yingyan.local/backend/internal/platform/database"
	"yingyan.local/backend/internal/respond"
	"yingyan.local/backend/internal/storage"
)

type Checker struct {
	db    *gorm.DB
	store storage.Store
}

func NewChecker(db *gorm.DB, store storage.Store) *Checker {
	return &Checker{db: db, store: store}
}

func (h *Checker) Live(c *gin.Context) {
	respond.OK(c, map[string]string{"status": "ok"})
}

func (h *Checker) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	if err := database.Ping(ctx, h.db); err != nil {
		respond.Error(c, apierror.New(http.StatusServiceUnavailable, apierror.CodeInternal, "服务尚未就绪", map[string]string{"component": "database"}))
		return
	}
	var migrationCount int64
	if err := h.db.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migration.LatestVersion,
	).Scan(&migrationCount).Error; err != nil || migrationCount != 1 {
		respond.Error(c, apierror.New(http.StatusServiceUnavailable, apierror.CodeInternal, "服务尚未就绪", map[string]string{"component": "migration"}))
		return
	}
	if err := h.store.Ready(ctx); err != nil {
		respond.Error(c, apierror.New(http.StatusServiceUnavailable, apierror.CodeInternal, "服务尚未就绪", map[string]string{"component": "object_storage"}))
		return
	}
	respond.OK(c, map[string]string{"status": "ready"})
}
