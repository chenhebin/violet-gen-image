package handlers

import (
	"testing"
	"time"

	"gorm.io/datatypes"

	"yingyan.local/backend/internal/model"
)

func TestModelDTOIncludesSafeTestSummary(t *testing.T) {
	testedAt := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	value := model.AIModel{
		ProviderID:      "provider-1",
		DisplayName:     "GPT Image 2",
		ModelID:         "gpt-image-2",
		Type:            "image",
		Capabilities:    datatypes.JSON([]byte(`{"textToImage":true}`)),
		Enabled:         true,
		TestStatus:      "error",
		LastTestedAt:    &testedAt,
		LastTestSummary: "模型测试失败：provider generate_image failed: timeout",
	}
	value.ID = "model-1"

	dto := modelDTO(value, "Daidai", map[string]*string{})
	lastTest, ok := dto["lastTest"].(map[string]any)
	if !ok {
		t.Fatalf("lastTest = %#v", dto["lastTest"])
	}
	if success, _ := lastTest["success"].(bool); success {
		t.Fatal("failed test must not be reported as successful")
	}
	if message := lastTest["message"]; message != value.LastTestSummary {
		t.Fatalf("message = %v", message)
	}
}
