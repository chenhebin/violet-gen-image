package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeliveryDigestIsStableAndContentSensitive(t *testing.T) {
	t.Parallel()

	first := multipartFileHeader(t, "result.png", []byte("first-image"))
	same := multipartFileHeader(t, "result.png", []byte("first-image"))
	changed := multipartFileHeader(t, "result.png", []byte("second-image"))

	firstDigest, err := deliveryDigest([]*multipart.FileHeader{first})
	if err != nil {
		t.Fatal(err)
	}
	sameDigest, err := deliveryDigest([]*multipart.FileHeader{same})
	if err != nil {
		t.Fatal(err)
	}
	changedDigest, err := deliveryDigest([]*multipart.FileHeader{changed})
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != sameDigest {
		t.Fatalf("same upload produced different digests: %q != %q", firstDigest, sameDigest)
	}
	if firstDigest == changedDigest {
		t.Fatal("different upload content produced the same digest")
	}
}

func multipartFileHeader(t *testing.T, filename string, data []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(int64(body.Len())); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if request.MultipartForm != nil {
			_ = request.MultipartForm.RemoveAll()
		}
	})
	return request.MultipartForm.File["files"][0]
}
