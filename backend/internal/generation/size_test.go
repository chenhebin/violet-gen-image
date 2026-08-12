package generation

import (
	"fmt"
	"math"
	"testing"
)

func TestSizeForSourceImageKeepsValidGPTImage2Size(t *testing.T) {
	if got := SizeForSourceImage("gpt-image-2", 800, 1200, "1:1"); got != "800x1200" {
		t.Fatalf("SizeForSourceImage() = %q, want %q", got, "800x1200")
	}
}

func TestSizeForSourceImageNormalizesSmallSource(t *testing.T) {
	got := SizeForSourceImage("gpt-image-2", 600, 800, "1:1")
	if got != "720x960" {
		t.Fatalf("SizeForSourceImage() = %q, want %q", got, "720x960")
	}
	assertValidCustomImageSize(t, got, 3.0/4.0)
}

func TestSizeForSourceImageNormalizesLargeSource(t *testing.T) {
	got := SizeForSourceImage("gpt-image-2", 3024, 4032, "1:1")
	if got != "2448x3264" {
		t.Fatalf("SizeForSourceImage() = %q, want %q", got, "2448x3264")
	}
	assertValidCustomImageSize(t, got, 3.0/4.0)
}

func TestSizeForSourceImageFallsBackForOtherModelsOrMissingSource(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		width, height int
	}{
		{name: "legacy GPT Image", model: "gpt-image-1", width: 800, height: 1200},
		{name: "similar model name", model: "gpt-image-20", width: 800, height: 1200},
		{name: "missing source", model: "gpt-image-2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := SizeForSourceImage(test.model, test.width, test.height, "16:9"); got != "1536x1024" {
				t.Fatalf("SizeForSourceImage() = %q, want fallback %q", got, "1536x1024")
			}
		})
	}
}

func assertValidCustomImageSize(t *testing.T, value string, wantRatio float64) {
	t.Helper()
	var width, height int
	if _, err := fmt.Sscanf(value, "%dx%d", &width, &height); err != nil {
		t.Fatalf("invalid size %q: %v", value, err)
	}
	pixels := width * height
	if width%customImageSizeStep != 0 || height%customImageSizeStep != 0 ||
		width > customImageSizeMaxEdge || height > customImageSizeMaxEdge ||
		pixels < customImageSizeMinPixels || pixels > customImageSizeMaxPixels ||
		width > customImageSizeMaxRatio*height || height > customImageSizeMaxRatio*width {
		t.Fatalf("size %q violates GPT Image 2 constraints", value)
	}
	if ratio := float64(width) / float64(height); math.Abs(ratio-wantRatio) > 0.001 {
		t.Fatalf("size %q ratio = %f, want %f", value, ratio, wantRatio)
	}
}
