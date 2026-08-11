package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/respond"
)

type Handler struct {
	service *Service
	cfg     config.Config
}

func NewHandler(service *Service, cfg config.Config) *Handler {
	return &Handler{service: service, cfg: cfg}
}

func (h *Handler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("请求参数无效", nil))
		return
	}
	user, token, err := h.service.Register(c.Request.Context(), input, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		respond.Error(c, err)
		return
	}
	h.setSessionCookie(c, h.cfg.Auth.UserCookieName, token)
	respond.Created(c, user)
}

func (h *Handler) LoginUser(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("请求参数无效", nil))
		return
	}
	user, token, err := h.service.LoginUser(c.Request.Context(), input, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		respond.Error(c, err)
		return
	}
	h.setSessionCookie(c, h.cfg.Auth.UserCookieName, token)
	respond.OK(c, user)
}

func (h *Handler) UserSession(c *gin.Context) {
	principal, ok := UserPrincipalFrom(c)
	if !ok {
		respond.Error(c, apierror.AuthRequired())
		return
	}
	respond.OK(c, principal.User)
}

func (h *Handler) LogoutUser(c *gin.Context) {
	principal, ok := UserPrincipalFrom(c)
	if !ok {
		respond.Error(c, apierror.AuthRequired())
		return
	}
	if err := h.service.LogoutUser(c.Request.Context(), principal.SessionID); err != nil {
		respond.Error(c, err)
		return
	}
	h.clearSessionCookie(c, h.cfg.Auth.UserCookieName)
	respond.NoData(c)
}

func (h *Handler) LoginAdmin(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("请求参数无效", nil))
		return
	}
	admin, token, err := h.service.LoginAdmin(c.Request.Context(), input, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		respond.Error(c, err)
		return
	}
	h.setSessionCookie(c, h.cfg.Auth.AdminCookieName, token)
	respond.OK(c, admin)
}

func (h *Handler) AdminSession(c *gin.Context) {
	principal, ok := AdminPrincipalFrom(c)
	if !ok {
		respond.Error(c, apierror.AuthRequired())
		return
	}
	respond.OK(c, principal.Admin)
}

func (h *Handler) LogoutAdmin(c *gin.Context) {
	principal, ok := AdminPrincipalFrom(c)
	if !ok {
		respond.Error(c, apierror.AuthRequired())
		return
	}
	if err := h.service.LogoutAdmin(c.Request.Context(), principal.SessionID); err != nil {
		respond.Error(c, err)
		return
	}
	h.clearSessionCookie(c, h.cfg.Auth.AdminCookieName)
	respond.NoData(c)
}

func (h *Handler) setSessionCookie(c *gin.Context, name string, token SessionToken) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    token.Raw,
		Path:     "/api",
		Domain:   h.cfg.Security.CookieDomain,
		HttpOnly: true,
		Secure:   h.cfg.Security.CookieSecure,
		SameSite: sameSite(h.cfg.Security.CookieSameSite),
	}
	if token.Remember {
		cookie.Expires = token.ExpiresAt
		cookie.MaxAge = max(1, int(time.Until(token.ExpiresAt).Seconds()))
	}
	http.SetCookie(c.Writer, cookie)
}

func (h *Handler) clearSessionCookie(c *gin.Context, name string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/api",
		Domain:   h.cfg.Security.CookieDomain,
		HttpOnly: true,
		Secure:   h.cfg.Security.CookieSecure,
		SameSite: sameSite(h.cfg.Security.CookieSameSite),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}

func sameSite(value string) http.SameSite {
	switch value {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
