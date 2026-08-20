package config

import (
	"strings"
	"testing"
)

func TestLoadReadsSeedAccountEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("CLIENT_USER_EMAIL", " User@Example.COM ")
	t.Setenv("CLIENT_USER_PASSWORD", "user-password")
	t.Setenv("PLATFORM_ADMIN_EMAIL", " Admin@Example.COM ")
	t.Setenv("RETOUCH_ADMIN_EMAIL", " Retouch@Example.COM ")
	t.Setenv("RETOUCH_ADMIN_PASSWORD", "retouch-password")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SeedAccounts.ClientUserEmail != "user@example.com" || cfg.SeedAccounts.ClientUserPassword != "user-password" {
		t.Fatalf("SeedAccounts user credentials = %#v", cfg.SeedAccounts)
	}
	if cfg.SeedAccounts.PlatformAdminEmail != "admin@example.com" {
		t.Fatalf("SeedAccounts platform admin = %#v", cfg.SeedAccounts)
	}
	if cfg.SeedAccounts.RetouchAdminEmail != "retouch@example.com" || cfg.SeedAccounts.RetouchAdminPassword != "retouch-password" {
		t.Fatalf("SeedAccounts retouch credentials = %#v", cfg.SeedAccounts)
	}
}

func TestLoadReadsPublicClaimConfiguration(t *testing.T) {
	t.Setenv("PUBLIC_WEB_URL", "https://img.example.com/")
	t.Setenv("CLIENT_PRODUCT_CODE", "image-client")
	t.Setenv("CLIENT_PRODUCT_NAME", "AI 图片 10 次")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.PublicWebURL != "https://img.example.com" ||
		cfg.App.ClientProductCode != "image-client" ||
		cfg.App.ClientProductName != "AI 图片 10 次" {
		t.Fatalf("App claim configuration = %#v", cfg.App)
	}
}

func TestSeedAccountsConfigValidate(t *testing.T) {
	t.Parallel()

	cfg := SeedAccountsConfig{
		ClientUserEmail:      "user@example.com",
		ClientUserPassword:   "user-password",
		PlatformAdminEmail:   "admin@example.com",
		RetouchAdminEmail:    "retouch@example.com",
		RetouchAdminPassword: "retouch-password",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSeedAccountsConfigValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	valid := SeedAccountsConfig{
		ClientUserEmail:      "user@example.com",
		ClientUserPassword:   "user-password",
		PlatformAdminEmail:   "admin@example.com",
		RetouchAdminEmail:    "retouch@example.com",
		RetouchAdminPassword: "retouch-password",
	}
	tests := []struct {
		name    string
		mutate  func(*SeedAccountsConfig)
		wantErr string
	}{
		{
			name:    "missing email",
			mutate:  func(cfg *SeedAccountsConfig) { cfg.ClientUserEmail = "" },
			wantErr: "CLIENT_USER_EMAIL is required",
		},
		{
			name:    "invalid email",
			mutate:  func(cfg *SeedAccountsConfig) { cfg.PlatformAdminEmail = "not-an-email" },
			wantErr: "PLATFORM_ADMIN_EMAIL must be a valid email address",
		},
		{
			name:    "duplicate email",
			mutate:  func(cfg *SeedAccountsConfig) { cfg.RetouchAdminEmail = cfg.PlatformAdminEmail },
			wantErr: "RETOUCH_ADMIN_EMAIL must differ from PLATFORM_ADMIN_EMAIL",
		},
		{
			name:    "short password",
			mutate:  func(cfg *SeedAccountsConfig) { cfg.RetouchAdminPassword = "short" },
			wantErr: "RETOUCH_ADMIN_PASSWORD must contain 8 to 72 bytes",
		},
		{
			name:    "missing password",
			mutate:  func(cfg *SeedAccountsConfig) { cfg.ClientUserPassword = "" },
			wantErr: "CLIENT_USER_PASSWORD is required",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			test.mutate(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error = %v, want containing %q", err, test.wantErr)
			}
		})
	}
}
