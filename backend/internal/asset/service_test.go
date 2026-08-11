package asset

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestInspectImageUsesDecodedContent(t *testing.T) {
	var encoded bytes.Buffer
	source := image.NewRGBA(image.Rect(0, 0, 3, 2))
	source.Set(0, 0, color.White)
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}

	contentType, width, height, err := inspectImage(encoded.Bytes(), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "image/png" || width != 3 || height != 2 {
		t.Fatalf("got %s %dx%d", contentType, width, height)
	}
}

func TestRejectsMismatchedContentType(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := inspectImage(encoded.Bytes(), "image/jpeg"); err == nil {
		t.Fatal("expected mismatched content type to fail")
	}
}
