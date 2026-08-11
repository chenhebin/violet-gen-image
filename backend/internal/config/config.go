package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Security SecurityConfig
	Auth     AuthConfig
	Worker   WorkerConfig
	Provider ProviderConfig
}

type AppConfig struct {
	Env            string
	HTTPAddr       string
	AllowedOrigins []string
	APIBasePath    string
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type StorageConfig struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	Region         string
	UseSSL         bool
}

type SecurityConfig struct {
	EncryptionKey    string
	TokenPepper      string
	RedemptionPepper string
	CookieSecure     bool
	CookieDomain     string
	CookieSameSite   string
	BcryptCost       int
}

type AuthConfig struct {
	UserCookieName   string
	AdminCookieName  string
	UserSessionTTL   time.Duration
	UserRememberTTL  time.Duration
	AdminSessionTTL  time.Duration
	AdminRememberTTL time.Duration
	TermsVersion     string
	AllowDemoSeed    bool
}

type WorkerConfig struct {
	HealthAddr   string
	PollInterval time.Duration
	StaleAfter   time.Duration
	WorkerID     string
}

type ProviderConfig struct {
	AllowHTTP             bool
	AllowPrivateNetwork   bool
	ConnectTimeout        time.Duration
	RequestTimeout        time.Duration
	ResponseHeaderTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Env:            env("APP_ENV", "development"),
			HTTPAddr:       env("HTTP_ADDR", ":8080"),
			AllowedOrigins: csvEnv("ALLOWED_ORIGINS", "http://127.0.0.1:5173,http://localhost:5173,http://127.0.0.1:5174,http://localhost:5174"),
			APIBasePath:    "/api",
		},
		Database: DatabaseConfig{
			URL:             env("DATABASE_URL", "postgres://yingyan:yingyan_dev@127.0.0.1:5432/yingyan?sslmode=disable"),
			MaxOpenConns:    intEnv("DB_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    intEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: durationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Storage: StorageConfig{
			Endpoint:       env("MINIO_ENDPOINT", "127.0.0.1:9000"),
			PublicEndpoint: env("MINIO_PUBLIC_ENDPOINT", "127.0.0.1:9000"),
			AccessKey:      env("MINIO_ACCESS_KEY", "yingyan"),
			SecretKey:      env("MINIO_SECRET_KEY", "yingyan_dev_secret"),
			Bucket:         env("MINIO_BUCKET", "yingyan-assets"),
			Region:         env("MINIO_REGION", "us-east-1"),
			UseSSL:         boolEnv("MINIO_USE_SSL", false),
		},
		Security: SecurityConfig{
			EncryptionKey:    env("ENCRYPTION_KEY", "dev-encryption-key-change-me-32-bytes"),
			TokenPepper:      env("TOKEN_PEPPER", "dev-token-pepper-change-me"),
			RedemptionPepper: env("REDEMPTION_PEPPER", "dev-redemption-pepper-change-me"),
			CookieSecure:     boolEnv("COOKIE_SECURE", false),
			CookieDomain:     strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")),
			CookieSameSite:   strings.ToLower(env("COOKIE_SAME_SITE", "lax")),
			BcryptCost:       intEnv("BCRYPT_COST", 12),
		},
		Auth: AuthConfig{
			UserCookieName:   env("USER_SESSION_COOKIE", "yy_user_session"),
			AdminCookieName:  env("ADMIN_SESSION_COOKIE", "yy_manage_session"),
			UserSessionTTL:   durationEnv("USER_SESSION_TTL", 24*time.Hour),
			UserRememberTTL:  durationEnv("USER_REMEMBER_TTL", 30*24*time.Hour),
			AdminSessionTTL:  durationEnv("ADMIN_SESSION_TTL", 8*time.Hour),
			AdminRememberTTL: durationEnv("ADMIN_REMEMBER_TTL", 7*24*time.Hour),
			TermsVersion:     env("TERMS_VERSION", "v0.1"),
			AllowDemoSeed:    boolEnv("ALLOW_DEMO_SEED", false),
		},
		Worker: WorkerConfig{
			HealthAddr:   env("WORKER_HEALTH_ADDR", ":8081"),
			PollInterval: durationEnv("WORKER_POLL_INTERVAL", time.Second),
			StaleAfter:   durationEnv("WORKER_STALE_AFTER", 5*time.Minute),
			WorkerID:     env("WORKER_ID", hostname()),
		},
		Provider: ProviderConfig{
			AllowHTTP:             boolEnv("PROVIDER_ALLOW_HTTP", false),
			AllowPrivateNetwork:   boolEnv("PROVIDER_ALLOW_PRIVATE_NETWORK", false),
			ConnectTimeout:        durationEnv("PROVIDER_CONNECT_TIMEOUT", 10*time.Second),
			RequestTimeout:        durationEnv("PROVIDER_REQUEST_TIMEOUT", 180*time.Second),
			ResponseHeaderTimeout: durationEnv("PROVIDER_RESPONSE_HEADER_TIMEOUT", 180*time.Second),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Database.URL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if c.Storage.Endpoint == "" || c.Storage.AccessKey == "" || c.Storage.SecretKey == "" || c.Storage.Bucket == "" {
		return errors.New("MinIO endpoint, credentials and bucket are required")
	}
	if len(c.Security.EncryptionKey) < 32 {
		return errors.New("ENCRYPTION_KEY must contain at least 32 characters")
	}
	if len(c.Security.TokenPepper) < 16 || len(c.Security.RedemptionPepper) < 16 {
		return errors.New("TOKEN_PEPPER and REDEMPTION_PEPPER must contain at least 16 characters")
	}
	if c.Security.BcryptCost < 10 || c.Security.BcryptCost > 15 {
		return errors.New("BCRYPT_COST must be between 10 and 15")
	}
	switch c.Security.CookieSameSite {
	case "lax", "strict", "none":
	default:
		return fmt.Errorf("unsupported COOKIE_SAME_SITE %q", c.Security.CookieSameSite)
	}
	if c.Security.CookieSameSite == "none" && !c.Security.CookieSecure {
		return errors.New("COOKIE_SECURE must be true when COOKIE_SAME_SITE=none")
	}
	if len(c.App.AllowedOrigins) == 0 {
		return errors.New("at least one ALLOWED_ORIGINS value is required")
	}
	if c.Worker.StaleAfter <= c.Provider.RequestTimeout {
		return errors.New("WORKER_STALE_AFTER must be greater than PROVIDER_REQUEST_TIMEOUT")
	}
	if strings.EqualFold(c.App.Env, "production") {
		if !c.Security.CookieSecure {
			return errors.New("COOKIE_SECURE must be true in production")
		}
		if c.Security.EncryptionKey == "dev-encryption-key-change-me-32-bytes" ||
			c.Security.TokenPepper == "dev-token-pepper-change-me" ||
			c.Security.RedemptionPepper == "dev-redemption-pepper-change-me" {
			return errors.New("development encryption keys and peppers cannot be used in production")
		}
		if c.Storage.SecretKey == "yingyan_dev_secret" {
			return errors.New("development object storage credentials cannot be used in production")
		}
	}
	return nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func csvEnv(key, fallback string) []string {
	raw := env(key, fallback)
	values := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if value := strings.TrimSpace(item); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func hostname() string {
	value, err := os.Hostname()
	if err != nil || value == "" {
		return "yingyan-worker"
	}
	return value
}
