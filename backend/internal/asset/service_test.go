package asset

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/url"
	"testing"
	"time"

	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/storage"
)

type signingStore struct{}

func (signingStore) Bucket() string                     { return "test" }
func (signingStore) EnsureBucket(context.Context) error { return nil }
func (signingStore) Ready(context.Context) error        { return nil }
func (signingStore) Put(context.Context, string, io.Reader, int64, storage.PutOptions) (storage.ObjectInfo, error) {
	return storage.ObjectInfo{}, nil
}
func (signingStore) Get(context.Context, string) (io.ReadCloser, storage.ObjectInfo, error) {
	return nil, storage.ObjectInfo{}, nil
}
func (signingStore) Delete(context.Context, string) error { return nil }
func (signingStore) PresignGet(_ context.Context, key string, _ time.Duration, _ storage.PresignOptions) (*url.URL, error) {
	return url.Parse("https://storage.test/" + key)
}

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

func TestSignedURLValidatesPurposeAndExpiry(t *testing.T) {
	service := New(nil, signingStore{})
	value := model.Asset{ObjectKey: "users/u/result.png", MIMEType: "image/png", OriginalName: "result.png"}
	signed, err := service.SignedURL(context.Background(), value, "preview")
	if err != nil {
		t.Fatal(err)
	}
	if signed.URL == "" || signed.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("signed URL = %#v", signed)
	}
	if _, err := service.SignedURL(context.Background(), value, "unknown"); err == nil {
		t.Fatal("expected invalid purpose to fail")
	}
}
