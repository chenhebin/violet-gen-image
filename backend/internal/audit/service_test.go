package audit

import "testing"

func TestSanitizeNestedSecrets(t *testing.T) {
	value := Sanitize(map[string]any{
		"provider": map[string]any{"apiKey": "secret", "name": "test"},
		"password": "secret",
	}).(map[string]any)
	if value["password"] != "[REDACTED]" {
		t.Fatal("password was not redacted")
	}
	provider := value["provider"].(map[string]any)
	if provider["apiKey"] != "[REDACTED]" || provider["name"] != "test" {
		t.Fatalf("unexpected provider snapshot %#v", provider)
	}
}
