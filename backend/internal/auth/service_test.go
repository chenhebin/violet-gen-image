package auth

import (
	"reflect"
	"testing"

	"yingyan.local/backend/internal/model"
)

func TestPermissionsForRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role string
		want []string
	}{
		{
			role: model.AdminRolePlatformAdmin,
			want: []string{PermissionPlatformManage, PermissionRetouchManage},
		},
		{
			role: model.AdminRoleRetouchOperator,
			want: []string{PermissionRetouchManage},
		},
		{role: "unknown", want: []string{}},
	}
	for _, test := range tests {
		if got := PermissionsForRole(test.role); !reflect.DeepEqual(got, test.want) {
			t.Fatalf("PermissionsForRole(%q) = %v, want %v", test.role, got, test.want)
		}
	}
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()
	email, err := validateCredentials(" DEMO@YINGYAN.LOCAL ", "Demo1234!")
	if err != nil {
		t.Fatalf("validateCredentials() error = %v", err)
	}
	if email != "demo@yingyan.local" {
		t.Fatalf("validateCredentials() email = %q", email)
	}
	if _, err := validateCredentials("invalid", "Demo1234!"); err == nil {
		t.Fatal("validateCredentials() accepted invalid email")
	}
	if _, err := validateCredentials("demo@yingyan.local", "short"); err == nil {
		t.Fatal("validateCredentials() accepted short password")
	}
}
