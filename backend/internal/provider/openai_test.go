package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var onePixelPNG = mustDecodeBase64(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
)

func TestOptimizePromptWithImageContext(t *testing.T) {
	t.Parallel()
	requests := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" {
			http.NotFound(writer, request)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(request.Body).Decode(&body)
		requests <- body
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Request-Id", "req-chat-1")
		_, _ = io.WriteString(writer, `{
			"model":"chat-model",
			"choices":[{"message":{"content":"优化后的提示词"}}],
			"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
		}`)
	}))
	defer server.Close()

	adapter := newTestAdapter(t, server.URL+"/v1", Config{})
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(onePixelPNG)
	result, err := adapter.OptimizePrompt(context.Background(), OptimizePromptRequest{
		Model:         "chat-model",
		SystemPrompt:  "system",
		Prompt:        "improve this",
		ImageDataURLs: []string{dataURL},
		MaxTokens:     200,
	})
	if err != nil {
		t.Fatalf("OptimizePrompt() error = %v", err)
	}
	if result.Content != "优化后的提示词" || result.RequestID != "req-chat-1" {
		t.Fatalf("OptimizePrompt() result = %#v", result)
	}
	if result.Usage.TotalTokens != 15 {
		t.Fatalf("Usage = %#v", result.Usage)
	}

	requestBody := <-requests
	if requestBody["model"] != "chat-model" {
		t.Fatalf("model = %#v", requestBody["model"])
	}
	if stream, exists := requestBody["stream"]; !exists || stream != false {
		t.Fatalf("stream = %#v, want false", stream)
	}
	messages := requestBody["messages"].([]any)
	userMessage := messages[1].(map[string]any)
	content := userMessage["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("user content length = %d", len(content))
	}
	imageContent := content[1].(map[string]any)
	imageURL := imageContent["image_url"].(map[string]any)
	if imageURL["url"] != dataURL {
		t.Fatal("image data URL was not forwarded")
	}
}

func TestGenerateTextToImageParsesBase64AndSignedURL(t *testing.T) {
	t.Parallel()
	var server *httptest.Server
	requests := make(chan map[string]any, 1)
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/images/generations":
			var body map[string]any
			_ = json.NewDecoder(request.Body).Decode(&body)
			requests <- body
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"data": []map[string]string{
					{
						"b64_json":       base64.StdEncoding.EncodeToString(onePixelPNG),
						"revised_prompt": "first",
					},
					{
						"url":            server.URL + "/generated.png?signature=secret",
						"revised_prompt": "second",
					},
				},
			})
		case "/generated.png":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write(onePixelPNG)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	adapter := newTestAdapter(t, server.URL, Config{})
	images, err := adapter.GenerateTextToImage(context.Background(), TextToImageRequest{
		Model:       "image-model",
		Prompt:      "portrait",
		OutputCount: 2,
		Size:        "800x1200",
	})
	if err != nil {
		t.Fatalf("GenerateTextToImage() error = %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("image count = %d", len(images))
	}
	if images[0].Source != ImageSourceBase64 || images[1].Source != ImageSourceURL {
		t.Fatalf("image sources = %q, %q", images[0].Source, images[1].Source)
	}
	for index, image := range images {
		if image.ContentType != "image/png" {
			t.Fatalf("image %d content type = %q", index, image.ContentType)
		}
		if string(image.Data) != string(onePixelPNG) {
			t.Fatalf("image %d data does not match", index)
		}
	}
	requestBody := <-requests
	if requestBody["n"] != float64(2) {
		t.Fatalf("multi-image request n = %#v, want 2", requestBody["n"])
	}
	if requestBody["size"] != "800x1200" {
		t.Fatalf("image request size = %#v, want %q", requestBody["size"], "800x1200")
	}
	if _, exists := requestBody["response_format"]; exists {
		t.Fatalf("image request unexpectedly included response_format: %#v", requestBody["response_format"])
	}
	if _, exists := requestBody["user"]; exists {
		t.Fatalf("image request unexpectedly included user: %#v", requestBody["user"])
	}
}

func TestGenerateImageToImageStreamsMultipart(t *testing.T) {
	t.Parallel()
	type captured struct {
		model      string
		prompt     string
		size       string
		imageCount int
		imageData  []byte
	}
	requests := make(chan captured, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseMultipartForm(2 << 20); err != nil {
			http.Error(writer, "invalid multipart", http.StatusBadRequest)
			return
		}
		files := request.MultipartForm.File["image[]"]
		var imageData []byte
		if len(files) > 0 {
			file, _ := files[0].Open()
			imageData, _ = io.ReadAll(file)
			_ = file.Close()
		}
		requests <- captured{
			model:      request.FormValue("model"),
			prompt:     request.FormValue("prompt"),
			size:       request.FormValue("size"),
			imageCount: len(files),
			imageData:  imageData,
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"data": []map[string]string{
				{"b64_json": base64.StdEncoding.EncodeToString(onePixelPNG)},
			},
		})
	}))
	defer server.Close()

	adapter := newTestAdapter(t, server.URL, Config{})
	images, err := adapter.GenerateImageToImage(context.Background(), ImageToImageRequest{
		Model:  "image-model",
		Prompt: "retouch",
		Images: []ImageInput{
			{
				Filename:    "source.png",
				ContentType: "image/png",
				Reader:      strings.NewReader(string(onePixelPNG)),
			},
		},
		OutputCount: 1,
		Size:        "800x1200",
	})
	if err != nil {
		t.Fatalf("GenerateImageToImage() error = %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("image count = %d", len(images))
	}
	capturedRequest := <-requests
	if capturedRequest.model != "image-model" ||
		capturedRequest.prompt != "retouch" ||
		capturedRequest.size != "800x1200" ||
		capturedRequest.imageCount != 1 ||
		string(capturedRequest.imageData) != string(onePixelPNG) {
		t.Fatalf("multipart request = %#v", capturedRequest)
	}
}

func TestProviderErrorDoesNotExposeResponseOrAPIKey(t *testing.T) {
	t.Parallel()
	const secret = "sk-very-secret"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(writer, `{
			"error":{
				"message":"failed with sk-very-secret at http://internal.local",
				"code":"invalid_request"
			}
		}`)
	}))
	defer server.Close()

	adapter := newTestAdapter(t, server.URL, Config{APIKey: secret})
	_, err := adapter.OptimizePrompt(context.Background(), OptimizePromptRequest{
		Model:  "chat-model",
		Prompt: "hello",
	})
	if err == nil {
		t.Fatal("OptimizePrompt() unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "internal.local") {
		t.Fatalf("error leaked provider details: %v", err)
	}
	var providerErr *Error
	if !errors.As(err, &providerErr) {
		t.Fatalf("error type = %T", err)
	}
	if providerErr.ProviderCode != "invalid_request" || providerErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("provider error = %#v", providerErr)
	}
}

func TestProviderResponseLimitAndCancellation(t *testing.T) {
	t.Parallel()
	t.Run("response size", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(writer, strings.Repeat("x", 128))
		}))
		defer server.Close()
		adapter := newTestAdapter(t, server.URL, Config{MaxResponseBytes: 64})
		_, err := adapter.TestConnection(context.Background())
		var providerErr *Error
		if !errors.As(err, &providerErr) || providerErr.Kind != ErrorResponseTooLarge {
			t.Fatalf("TestConnection() error = %#v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			<-request.Context().Done()
		}))
		defer server.Close()
		adapter := newTestAdapter(t, server.URL, Config{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := adapter.TestConnection(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("TestConnection() error = %v", err)
		}
	})
}

func TestClassifyTransportErrorRecognizesNetworkTimeout(t *testing.T) {
	t.Parallel()
	err := classifyTransportError("generate_image", &net.DNSError{IsTimeout: true})
	var providerErr *Error
	if !errors.As(err, &providerErr) {
		t.Fatalf("classifyTransportError() type = %T", err)
	}
	if providerErr.Kind != ErrorTimeout || !providerErr.Retryable {
		t.Fatalf("classifyTransportError() = %#v", providerErr)
	}
	if !errors.Is(providerErr, context.DeadlineExceeded) {
		t.Fatalf("classifyTransportError() cause = %v", providerErr.Unwrap())
	}
}

func TestFetchImageRejectsPrivateTargetByDefault(t *testing.T) {
	t.Parallel()
	adapter := &OpenAICompatible{
		client:        http.DefaultClient,
		policy:        OutboundPolicy{},
		maxImageBytes: defaultMaxImageBytes,
	}
	_, _, err := adapter.fetchImage(context.Background(), "http://127.0.0.1/image.png")
	var providerErr *Error
	if !errors.As(err, &providerErr) || providerErr.Kind != ErrorUnsafeURL {
		t.Fatalf("fetchImage() error = %#v", err)
	}
}

func TestValidateImageDataURLRejectsSVG(t *testing.T) {
	t.Parallel()
	value := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte("<svg/>"))
	if err := validateImageDataURL(value, 1024); err == nil {
		t.Fatal("validateImageDataURL() unexpectedly accepted SVG")
	}
}

func newTestAdapter(t *testing.T, baseURL string, override Config) *OpenAICompatible {
	t.Helper()
	cfg := Config{
		BaseURL:             baseURL,
		APIKey:              "sk-test",
		RequestTimeout:      3 * time.Second,
		AllowHTTP:           true,
		AllowPrivateNetwork: true,
	}
	if override.APIKey != "" {
		cfg.APIKey = override.APIKey
	}
	if override.MaxResponseBytes != 0 {
		cfg.MaxResponseBytes = override.MaxResponseBytes
	}
	adapter, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible() error = %v", err)
	}
	return adapter
}

func mustDecodeBase64(value string) []byte {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return data
}
