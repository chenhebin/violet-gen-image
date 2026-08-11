package storage

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		defaultSSL bool
		host       string
		secure     bool
		wantError  bool
	}{
		{name: "host and port", input: "minio:9000", host: "minio:9000"},
		{name: "host and port SSL", input: "minio:9000", defaultSSL: true, host: "minio:9000", secure: true},
		{name: "HTTP URL", input: "http://127.0.0.1:9000", host: "127.0.0.1:9000"},
		{name: "HTTPS URL", input: "https://s3.example.com", host: "s3.example.com", secure: true},
		{name: "path", input: "https://s3.example.com/storage", wantError: true},
		{name: "credentials", input: "https://user:secret@s3.example.com", wantError: true},
		{name: "unsupported scheme", input: "ftp://s3.example.com", wantError: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			host, secure, err := normalizeEndpoint(test.input, test.defaultSSL)
			if (err != nil) != test.wantError {
				t.Fatalf("normalizeEndpoint() error = %v, wantError = %v", err, test.wantError)
			}
			if host != test.host || secure != test.secure {
				t.Fatalf("normalizeEndpoint() = (%q, %v)", host, secure)
			}
		})
	}
}

func TestValidateObjectKey(t *testing.T) {
	t.Parallel()
	for _, key := range []string{
		"users/user-id/source/asset-id.png",
		"ai-results/2026/07/asset.webp",
	} {
		if err := validateObjectKey(key); err != nil {
			t.Fatalf("validateObjectKey(%q) error = %v", key, err)
		}
	}
	for _, key := range []string{
		"",
		"/absolute.png",
		"../secret",
		"assets/../secret",
		"assets\\secret",
		"assets//image.png",
		"assets/\nimage.png",
		strings.Repeat("a", 1025),
	} {
		if err := validateObjectKey(key); err == nil {
			t.Fatalf("validateObjectKey(%q) unexpectedly succeeded", key)
		}
	}
}

func TestContentDispositionPreventsHeaderInjection(t *testing.T) {
	t.Parallel()
	value := contentDisposition("result.jpg\r\nX-Injected: true")
	if strings.ContainsAny(value, "\r\n") {
		t.Fatalf("contentDisposition() contains a newline: %q", value)
	}
	if !strings.HasPrefix(value, "attachment") {
		t.Fatalf("contentDisposition() = %q", value)
	}
}

func TestPresignDurationBounds(t *testing.T) {
	t.Parallel()
	if minPresignDuration != time.Minute || maxPresignDuration != 24*time.Hour {
		t.Fatalf("unexpected presign bounds: %s to %s", minPresignDuration, maxPresignDuration)
	}
}
