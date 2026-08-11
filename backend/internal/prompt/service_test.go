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
