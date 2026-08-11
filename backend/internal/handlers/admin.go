package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/aiconfig"
	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/audit"
	"yingyan.local/backend/internal/auth"
	"yingyan.local/backend/internal/manage"
	"yingyan.local/backend/internal/redemption"
	"yingyan.local/backend/internal/respond"
	"yingyan.local/backend/internal/retouch"
)

type AdminHandler struct {
	manage      *manage.Service
	redemptions *redemption.Service
	ai          *aiconfig.Service
	retouches   *retouch.Service
	assets      *asset.Service
	audits      *audit.Service
}

func NewAdminHandler(
	manageService *manage.Service,
	redemptions *redemption.Service,
	aiService *aiconfig.Service,
	retouches *retouch.Service,
	assets *asset.Service,
	audits *audit.Service,
) *AdminHandler {
	return &AdminHandler{
		manage: manageService, redemptions: redemptions, ai: aiService,
		retouches: retouches, assets: assets, audits: audits,
	}
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	principal, _ := auth.AdminPrincipalFrom(c)
	value, err := h.manage.Dashboard(c.Request.Context(), principal.Admin.Role == "retouch_operator")
	write(c, value, err)
}

func (h *AdminHandler) audit(
	c *gin.Context,
	action string,
	resourceType string,
	resourceID string,
	reason string,
	after any,
	actionErr error,
) {
	principal, ok := auth.AdminPrincipalFrom(c)
	if !ok {
		return
	}
	var resourceIDPointer *string
	if resourceID != "" {
		resourceIDPointer = &resourceID
	}
	result := "success"
	if actionErr != nil {
		result = "failure"
	}
	adminID := principal.Admin.ID
	_ = h.audits.Record(context.WithoutCancel(c.Request.Context()), audit.Entry{
		AdminID: &adminID, AdminEmail: principal.Admin.Email, AdminRole: principal.Admin.Role,
		Action: action, ResourceType: resourceType, ResourceID: resourceIDPointer,
		After: after, Reason: reason, Result: result, RequestID: requestID(c),
		IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(),
	})
}

func pageQuery(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	return page, pageSize
}

func keywordQuery(c *gin.Context) string {
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		return keyword
	}
	return strings.TrimSpace(c.Query("search"))
}

func optionalBool(c *gin.Context, key string) *bool {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &value
}

func optionalInt(c *gin.Context, key string) *int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &value
}

func optionalTime(c *gin.Context, key string) *time.Time {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &value
}

func requestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}

func bindJSON(c *gin.Context, target any) bool {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		respond.Error(c, apierror.Invalid("请求参数无效", nil))
		return false
	}
	return true
}
