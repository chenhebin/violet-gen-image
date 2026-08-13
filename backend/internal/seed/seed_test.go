package seed

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"yingyan.local/backend/internal/config"
)

func TestHashPasswordsUsesConfiguredCredentials(t *testing.T) {
	t.Parallel()

	credentials := config.SeedAccountsConfig{
		ClientUserPassword:   "configured-user-password",
		RetouchAdminPassword: "configured-retouch-password",
	}
	hashes, err := hashPasswords(credentials, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hashPasswords() error = %v", err)
	}

	tests := []struct {
		name     string
		hash     string
		password string
	}{
		{"user", hashes.user, credentials.ClientUserPassword},
		{"retouch", hashes.retouch, credentials.RetouchAdminPassword},
	}
	for _, test := range tests {
		if err := bcrypt.CompareHashAndPassword([]byte(test.hash), []byte(test.password)); err != nil {
			t.Errorf("%s password hash does not match configured password: %v", test.name, err)
		}
	}
}
