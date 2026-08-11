package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/respond"
)

type rateWindow struct {
	start time.Time
	count int
}

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateWindow
	limit   int
	window  time.Duration
	now     func() time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]rateWindow),
		limit:   limit,
		window:  window,
		now:     time.Now,
	}
}

func (r *RateLimiter) Middleware(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := r.now()
		entryKey := key(c)

		r.mu.Lock()
		entry := r.entries[entryKey]
		if entry.start.IsZero() || now.Sub(entry.start) >= r.window {
			entry = rateWindow{start: now}
		}
		entry.count++
		r.entries[entryKey] = entry
		allowed := entry.count <= r.limit
		if len(r.entries) > 10_000 {
			r.removeExpired(now)
		}
		r.mu.Unlock()

		if !allowed {
			respond.Error(c, apierror.New(429, apierror.CodeRateLimited, "请求过于频繁，请稍后重试", nil))
			return
		}
		c.Next()
	}
}

func (r *RateLimiter) removeExpired(now time.Time) {
	for key, entry := range r.entries {
		if now.Sub(entry.start) >= r.window {
			delete(r.entries, key)
		}
	}
}
