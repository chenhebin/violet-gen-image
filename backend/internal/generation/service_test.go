package generation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gorm.io/datatypes"

	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/prompt"
)

func TestTerminalState(t *testing.T) {
	now := time.Now()
	tests := []struct {
		completed int
		failed    int
		total     int
		want      string
	}{
		{0, 0, 4, StatusProcessing},
		{4, 0, 4, StatusCompleted},
		{2, 2, 4, StatusPartial},
		{0, 4, 4, StatusFailed},
	}
	for _, test := range tests {
		got, _ := terminalState(test.completed, test.failed, test.total, now)
		if got != test.want {
			t.Fatalf("terminalState(%d,%d,%d) = %q, want %q", test.completed, test.failed, test.total, got, test.want)
		}
	}
}

func TestGenerationProviderPromptAppliesReferenceStrengthOnlyToImageMode(t *testing.T) {
	textPrompt := generationProviderPrompt("base", "text-to-image", 80)
	if textPrompt != "base" {
		t.Fatalf("text-to-image prompt changed: %q", textPrompt)
	}
	imagePrompt := generationProviderPrompt("base", "image-to-image", 80)
	if imagePrompt == "base" || !strings.Contains(imagePrompt, "80/100") {
		t.Fatalf("reference strength missing from image prompt: %q", imagePrompt)
	}
}

func TestSettingsValidation(t *testing.T) {
	valid := Settings{AspectRatio: "1:1", OutputCount: 4, ReferenceStrength: 50}
	if err := validateSettings(valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.OutputCount = 5
	if err := validateSettings(invalid); err == nil {
		t.Fatal("expected invalid output count")
	}
}

func TestPromptReferenceRolesUsesConfirmedSnapshot(t *testing.T) {
	version := promptVersionWithReferences(
		`[{"assetId":"asset-a","role":"composition"},{"assetId":"asset-b","role":"detail"}]`,
	)
	roles, err := promptReferenceRoles(version)
	if err != nil {
		t.Fatal(err)
	}
	if roles["asset-a"] != "composition" || roles["asset-b"] != "detail" {
		t.Fatalf("unexpected reference roles: %#v", roles)
	}
}

func TestGenerationModeForAssets(t *testing.T) {
	tests := []struct {
		name   string
		assets []model.Asset
		want   string
	}{
		{name: "text only", assets: nil, want: "text-to-image"},
		{
			name:   "source image",
			assets: []model.Asset{{Kind: asset.KindSource}},
			want:   "image-to-image",
		},
		{
			name:   "reference image only",
			assets: []model.Asset{{Kind: asset.KindReference}},
			want:   "image-to-image",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := generationModeForAssets(test.assets); got != test.want {
				t.Fatalf("generationModeForAssets() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestHasSourceAsset(t *testing.T) {
	if hasSourceAsset([]model.Asset{{Kind: asset.KindReference}}) {
		t.Fatal("reference image must not satisfy source image requirement")
	}
	if !hasSourceAsset([]model.Asset{{Kind: asset.KindSource}}) {
		t.Fatal("source image should satisfy source image requirement")
	}
}

func TestBuildDirectPromptVersionSnapshotsSourceAssetsAndReferenceRoles(t *testing.T) {
	now := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	version, roles, err := buildDirectPromptVersion(
		"user-1",
		"小猫四爪朝天，躺在地上",
		[]prompt.ReferenceAsset{{AssetID: "reference-1", Role: "style"}},
		[]model.Asset{
			{BaseModel: model.BaseModel{ID: "source-1"}, Kind: asset.KindSource},
			{BaseModel: model.BaseModel{ID: "reference-1"}, Kind: asset.KindReference},
		},
		"image-to-image",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if version.Source != "小猫四爪朝天，躺在地上" || version.Status != prompt.StatusConfirmed {
		t.Fatalf("unexpected direct prompt version: %#v", version)
	}
	if version.ConfirmedAt == nil || !version.ConfirmedAt.Equal(now) {
		t.Fatalf("confirmedAt = %#v, want %s", version.ConfirmedAt, now)
	}
	if roles["reference-1"] != "style" {
		t.Fatalf("reference roles = %#v", roles)
	}
	var sourceIDs []string
	if err := json.Unmarshal(version.SourceAssetIDs, &sourceIDs); err != nil {
		t.Fatal(err)
	}
	if len(sourceIDs) != 1 || sourceIDs[0] != "source-1" {
		t.Fatalf("source ids = %#v", sourceIDs)
	}
}

func TestBuildDirectPromptVersionRequiresEveryReferenceRole(t *testing.T) {
	_, _, err := buildDirectPromptVersion(
		"user-1",
		"生成一张参考风格的人像",
		nil,
		[]model.Asset{{
			BaseModel: model.BaseModel{ID: "reference-1"},
			Kind:      asset.KindReference,
		}},
		"image-to-image",
		time.Now().UTC(),
	)
	if err == nil {
		t.Fatal("expected missing reference role to be rejected")
	}
}

func promptVersionWithReferences(references string) model.PromptVersion {
	return model.PromptVersion{ReferenceAssets: datatypes.JSON([]byte(references))}
}
