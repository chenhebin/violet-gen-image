package bootstrap

import "testing"

func TestValidateAdminInput(t *testing.T) {
	t.Parallel()

	input, err := validateAdminInput(AdminInput{
		Email:    " Owner@Yingyan.Example ",
		Password: "long-production-password",
		Name:     " 平台负责人 ",
	})
	if err != nil {
		t.Fatalf("validateAdminInput() error = %v", err)
	}
	if input.Email != "owner@yingyan.example" {
		t.Fatalf("Email = %q", input.Email)
	}
	if input.Name != "平台负责人" {
		t.Fatalf("Name = %q", input.Name)
	}
}

func TestValidateAdminInputRejectsWeakValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input AdminInput
	}{
		{
			name:  "invalid email",
			input: AdminInput{Email: "invalid", Password: "long-production-password", Name: "管理员"},
		},
		{
			name:  "short password",
			input: AdminInput{Email: "owner@example.com", Password: "short", Name: "管理员"},
		},
		{
			name:  "empty name",
			input: AdminInput{Email: "owner@example.com", Password: "long-production-password"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := validateAdminInput(test.input); err == nil {
				t.Fatal("validateAdminInput() error = nil")
			}
		})
	}
}
