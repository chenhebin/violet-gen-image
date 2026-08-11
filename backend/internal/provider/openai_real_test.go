package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRealOpenAICompatible(t *testing.T) {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("RUN_REAL_AI_TESTS")), "true") {
		t.Skip("set RUN_REAL_AI_TESTS=true to run real provider acceptance")
	}

	baseURL := requiredRealProviderEnv(t, "E2E_AI_BASE_URL")
	apiKey := requiredRealProviderEnv(t, "E2E_AI_API_KEY")
	chatModel := requiredRealProviderEnv(t, "E2E_CHAT_MODEL")
	imageModel := requiredRealProviderEnv(t, "E2E_IMAGE_MODEL")

	adapter, err := NewOpenAICompatible(Config{
		BaseURL:               baseURL,
		APIKey:                apiKey,
		ConnectTimeout:        15 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
		RequestTimeout:        4 * time.Minute,
		AllowPrivateNetwork: strings.EqualFold(
			strings.TrimSpace(os.Getenv("E2E_PROVIDER_ALLOW_PRIVATE_NETWORK")),
			"true",
		),
	})
	if err != nil {
		t.Fatalf("NewOpenAICompatible() error = %v", err)
	}

	t.Run("chat with image context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(onePixelPNG)
		result, callErr := adapter.OptimizePrompt(ctx, OptimizePromptRequest{
			Model:         chatModel,
			SystemPrompt:  "你是提示词检查助手，只需简短确认请求已收到。",
			Prompt:        "请用中文回复 OK。",
			ImageDataURLs: []string{dataURL},
			MaxTokens:     32,
		})
		if callErr != nil {
			t.Fatalf("OptimizePrompt() error = %v", callErr)
		}
		if strings.TrimSpace(result.Content) == "" {
			t.Fatal("OptimizePrompt() returned empty content")
		}
		if result.Usage.TotalTokens <= 0 {
			t.Fatalf("OptimizePrompt() usage = %#v", result.Usage)
		}
	})

	t.Run("text to image", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()

		images, callErr := adapter.GenerateTextToImage(ctx, TextToImageRequest{
			Model:       imageModel,
			Prompt:      "一个极简的深青色圆形图标，纯白背景，无文字",
			OutputCount: 1,
			Size:        "1024x1536",
		})
		if callErr != nil {
			t.Fatalf("GenerateTextToImage() error = %v", callErr)
		}
		if len(images) != 1 {
			t.Fatalf("GenerateTextToImage() image count = %d, want 1", len(images))
		}
		if images[0].Source != ImageSourceBase64 && images[0].Source != ImageSourceURL {
			t.Fatalf("GenerateTextToImage() source = %q", images[0].Source)
		}
		config, _, decodeErr := image.DecodeConfig(bytes.NewReader(images[0].Data))
		if decodeErr != nil {
			t.Fatalf("generated image decode error = %v", decodeErr)
		}
		if config.Width <= 0 || config.Height <= 0 {
			t.Fatalf("generated image dimensions = %dx%d", config.Width, config.Height)
		}
		t.Logf("generated image accepted: %dx%d, %s", config.Width, config.Height, images[0].ContentType)
	})

}

func requiredRealProviderEnv(t *testing.T, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		t.Fatalf("%s is required when RUN_REAL_AI_TESTS=true", key)
	}
	return value
}
