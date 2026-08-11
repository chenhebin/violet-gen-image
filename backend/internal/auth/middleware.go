package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/platform/security"
	"yingyan.local/backend/internal/respond"
)

const (
	UserPrincipalKey  = "user_principal"
	AdminPrincipalKey = "admin_principal"
)

func RequireUser(service *Service, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken, err := c.Cookie(cfg.Auth.UserCookieName)
		if err != nil {
			respond.Error(c, apierror.AuthRequired())
			return
		}
		principal, err := service.AuthenticateUser(c.Request.Context(), rawToken)
		if err != nil {
			respond.Error(c, err)
			return
		}
		c.Set(UserPrincipalKey, principal)
		c.Next()
	}
}

func RequireAdmin(service *Service, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken, err := c.Cookie(cfg.Auth.AdminCookieName)
		if err != nil {
			respond.Error(c, apierror.AuthRequired())
			return
		}
		principal, err := service.AuthenticateAdmin(c.Request.Context(), rawToken)
		if err != nil {
			respond.Error(c, err)
			return
		}
		c.Set(AdminPrincipalKey, principal)
		c.Next()
	}
}

func RequireAdminCSRF(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		principal, ok := AdminPrincipalFrom(c)
		if !ok {
			respond.Error(c, apierror.AuthRequired())
			return
		}
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" && !IsAllowedOrigin(origin, cfg.App.AllowedOrigins) {
			respond.Error(c, apierror.Forbidden())
			return
		}
		provided := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
		if provided == "" || !security.EqualDigest(principal.CSRFToken, provided) {
			respond.Error(c, apierror.Forbidden())
			return
		}
		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := AdminPrincipalFrom(c)
		if !ok {
			respond.Error(c, apierror.AuthRequired())
			return
		}
		if !HasPermission(principal.Admin, permission) {
			respond.Error(c, apierror.Forbidden())
			return
		}
		c.Next()
	}
}

func UserPrincipalFrom(c *gin.Context) (UserPrincipal, bool) {
	value, ok := c.Get(UserPrincipalKey)
	if !ok {
		return UserPrincipal{}, false
	}
	principal, ok := value.(UserPrincipal)
	return principal, ok
}

func AdminPrincipalFrom(c *gin.Context) (AdminPrincipal, bool) {
	value, ok := c.Get(AdminPrincipalKey)
	if !ok {
		return AdminPrincipal{}, false
	}
	principal, ok := value.(AdminPrincipal)
	return principal, ok
}
