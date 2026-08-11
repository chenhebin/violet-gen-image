package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), CORS([]string{"http://localhost:5173"}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
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
