package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-Id", "req-test-123")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if got := recorder.Header().Get("X-Request-Id"); got != "req-test-123" {
		t.Fatalf("X-Request-Id = %q", got)
	}
}

func TestRateLimiterSetsRetryAfterAndSkipsReads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(1, time.Minute)
	router := gin.New()
	router.Use(limiter.Middleware(func(*gin.Context) string { return "test" }))
	router.GET("/read", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/write", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/write", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first write status = %d", first.Code)
	}
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/write", nil))
	if second.Code != http.StatusTooManyRequests || second.Header().Get("Retry-After") != "60" {
		t.Fatalf("limited response = %d, retry-after=%q", second.Code, second.Header().Get("Retry-After"))
	}
	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/read", nil))
	if read.Code != http.StatusNoContent {
		t.Fatalf("read status = %d", read.Code)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), CORS([]string{"https://img.daidaiweb.cn"}))
	router.POST("/api/auth/login", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	request.Header.Set("Origin", "http://192.168.0.104:4005")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCORSAllowsConfiguredPublicOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), CORS([]string{"https://img.daidaiweb.cn"}))
	router.POST("/api/auth/login", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	request.Header.Set("Origin", "https://img.daidaiweb.cn")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://img.daidaiweb.cn" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", got)
	}
}

func TestRequireIdempotencyKeyOnlyChecksMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequireIdempotencyKey())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.POST("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	getRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusNoContent {
		t.Fatalf("GET status = %d, want %d", getRecorder.Code, http.StatusNoContent)
	}

	postRequest := httptest.NewRequest(http.MethodPost, "/", nil)
	postRecorder := httptest.NewRecorder()
	router.ServeHTTP(postRecorder, postRequest)
	if postRecorder.Code != http.StatusBadRequest {
		t.Fatalf("POST status = %d, want %d", postRecorder.Code, http.StatusBadRequest)
	}

	keyedRequest := httptest.NewRequest(http.MethodPost, "/", nil)
	keyedRequest.Header.Set("Idempotency-Key", "manage-test-key")
	keyedRecorder := httptest.NewRecorder()
	router.ServeHTTP(keyedRecorder, keyedRequest)
	if keyedRecorder.Code != http.StatusNoContent {
		t.Fatalf("keyed POST status = %d, want %d", keyedRecorder.Code, http.StatusNoContent)
	}
}
