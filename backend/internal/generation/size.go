package generation

import (
	"fmt"
	"math"
	"strings"
)

const (
	customImageSizeModelPrefix = "gpt-image-2"
	customImageSizeStep        = 16
	customImageSizeMaxEdge     = 3840
	customImageSizeMinPixels   = 655_360
	customImageSizeMaxPixels   = 8_294_400
	customImageSizeMaxRatio    = 3
)

// SizeForSourceImage keeps the first source image's canvas whenever the active
// model supports arbitrary sizes. Invalid source sizes are scaled to the
// closest valid canvas while preserving their aspect ratio as closely as possible.
func SizeForSourceImage(modelID string, width, height int, fallbackAspectRatio string) string {
	fallback := SizeForAspectRatio(fallbackAspectRatio)
	if !supportsCustomImageSize(modelID) || width <= 0 || height <= 0 {
		return fallback
	}

	bestWidth, bestHeight := closestValidImageSize(width, height)
	if bestWidth == 0 || bestHeight == 0 {
		return fallback
	}
	return fmt.Sprintf("%dx%d", bestWidth, bestHeight)
}

func supportsCustomImageSize(modelID string) bool {
	normalized := strings.ToLower(strings.TrimSpace(modelID))
	return normalized == customImageSizeModelPrefix ||
		strings.HasPrefix(normalized, customImageSizeModelPrefix+"-")
}

func closestValidImageSize(sourceWidth, sourceHeight int) (int, int) {
	sourceRatio := float64(sourceWidth) / float64(sourceHeight)
	bestScore := math.Inf(1)
	bestDistance := math.MaxInt
	bestWidth, bestHeight := 0, 0

	for width := customImageSizeStep; width <= customImageSizeMaxEdge; width += customImageSizeStep {
		for height := customImageSizeStep; height <= customImageSizeMaxEdge; height += customImageSizeStep {
			pixels := width * height
			if pixels < customImageSizeMinPixels || pixels > customImageSizeMaxPixels {
				continue
			}
			if width > customImageSizeMaxRatio*height || height > customImageSizeMaxRatio*width {
				continue
			}

			candidateRatio := float64(width) / float64(height)
			aspectDistance := math.Abs(math.Log(candidateRatio / sourceRatio))
			scaleDistance := math.Abs(math.Log(float64(width)/float64(sourceWidth))) +
				math.Abs(math.Log(float64(height)/float64(sourceHeight)))
			score := aspectDistance*1000 + scaleDistance
			distance := absInt(width-sourceWidth) + absInt(height-sourceHeight)
			if score < bestScore-1e-12 || (math.Abs(score-bestScore) <= 1e-12 && distance < bestDistance) {
				bestScore = score
				bestDistance = distance
				bestWidth = width
				bestHeight = height
			}
		}
	}

	return bestWidth, bestHeight
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
