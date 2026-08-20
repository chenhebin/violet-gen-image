package prompt

import (
	"strings"
	"testing"
)

func TestPromptModeForImageCount(t *testing.T) {
	tests := []struct {
		name       string
		imageCount int
		want       string
	}{
		{name: "text only", imageCount: 0, want: "text-to-image"},
		{name: "one image", imageCount: 1, want: "image-to-image"},
		{name: "multiple images", imageCount: 3, want: "image-to-image"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := promptModeForImageCount(test.imageCount); got != test.want {
				t.Fatalf("promptModeForImageCount(%d) = %q, want %q", test.imageCount, got, test.want)
			}
		})
	}
}

func TestBuildGenerationPromptUsesRawRequirementWhenSectionsAreEmpty(t *testing.T) {
	result := BuildGenerationPrompt("小猫四爪朝天，躺在地上", Sections{})
	if result != "用户需求：小猫四爪朝天，躺在地上" {
		t.Fatalf("unexpected direct prompt: %q", result)
	}
	if strings.Contains(result, "主体：") {
		t.Fatalf("empty optimized sections leaked into direct prompt: %q", result)
	}
}

func TestBuildGenerationPromptKeepsReferencePromptTextual(t *testing.T) {
	result := BuildGenerationPrompt("只修改服装颜色", Sections{
		Subject:         "保留人物身份",
		ReferencePrompt: "清透杂志氛围，50mm 镜头，柔和侧逆光",
	})
	if !strings.Contains(result, "参考图提示词（仅用于风格、氛围和镜头参考，不替代原图主体）：") {
		t.Fatalf("reference prompt marker missing: %q", result)
	}
	if !strings.Contains(result, "清透杂志氛围") {
		t.Fatalf("reference prompt missing: %q", result)
	}
}

func TestParseReferencePromptAcceptsPlainTextAndJSON(t *testing.T) {
	plain := parseReferencePrompt("清透杂志氛围，柔和侧逆光")
	if plain != "清透杂志氛围，柔和侧逆光" {
		t.Fatalf("plain prompt = %q", plain)
	}
	jsonPrompt := parseReferencePrompt("```json\n{\"prompt\":\"50mm 人像镜头，真实材质\"}\n```")
	if jsonPrompt != "50mm 人像镜头，真实材质" {
		t.Fatalf("json prompt = %q", jsonPrompt)
	}
}
