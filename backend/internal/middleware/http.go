package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/respond"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-Id"))
		if len(requestID) < 8 || len(requestID) > 128 || strings.ContainsAny(requestID, "\r\n") {
			requestID = uuid.NewString()
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-Id", requestID)
		c.Next()
	}
}

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimRight(origin, "/")] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := strings.TrimRight(strings.TrimSpace(c.GetHeader("Origin")), "/")
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				respond.Error(c, apierror.Forbidden())
				return
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Accept,Content-Type,X-Request-Id,Idempotency-Key,X-CSRF-Token")
			c.Header("Access-Control-Expose-Headers", "X-Request-Id,Content-Disposition")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		code, _ := c.Get("business_code")
		logger.InfoContext(c.Request.Context(), "http_request",
			"request_id", RequestIDFrom(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"business_code", code,
			"latency_ms", time.Since(startedAt).Milliseconds(),
		)
	}
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(c.Request.Context(), "http_panic",
					"request_id", RequestIDFrom(c),
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
				)
				respond.Error(c, apierror.Internal(fmt.Errorf("panic: %v", recovered)))
			}
		}()
		c.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

func RequireIdempotencyKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if len(key) < 8 || len(key) > 128 || strings.ContainsAny(key, "\r\n") {
			respond.Error(c, apierror.Invalid("缺少或无效的 Idempotency-Key", nil))
			return
		}
		c.Next()
	}
}

func RequestIDFrom(c *gin.Context) string {
	value, _ := c.Get(RequestIDKey)
	requestID, _ := value.(string)
	return requestID
}
