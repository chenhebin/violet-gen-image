package aiconfig

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/datatypes"

	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/provider"
)

func TestPlatformImageCapabilities(t *testing.T) {
	err := validatePlatformCapabilities("image", Capabilities{
		TextToImage:  true,
		ImageToImage: false,
	})
	if err == nil {
		t.Fatal("expected incomplete image capabilities to fail")
	}
	if err := validatePlatformCapabilities("image", Capabilities{
		TextToImage:  true,
		ImageToImage: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestBaseURLRequiresHTTPS(t *testing.T) {
	if err := validateBaseURL("http://example.com", false); err == nil {
		t.Fatal("expected HTTP URL to fail")
	}
	if err := validateBaseURL("https://example.com/v1", false); err != nil {
		t.Fatal(err)
	}
}

func TestChatModelCapabilityTestIncludesVisionSample(t *testing.T) {
	capabilities, err := json.Marshal(Capabilities{
		PromptOptimization: true,
		VisionInput:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingAdapter{}
	err = testModelCapabilities(context.Background(), adapter, model.AIModel{
		ModelID:      "chat-model",
		Type:         "chat",
		Capabilities: datatypes.JSON(capabilities),
	})
	if err != nil {
		t.Fatal(err)
	}
	if adapter.optimizeRequest.Model != "chat-model" {
		t.Fatalf("model = %q", adapter.optimizeRequest.Model)
	}
	if len(adapter.optimizeRequest.ImageDataURLs) != 1 ||
		!strings.HasPrefix(adapter.optimizeRequest.ImageDataURLs[0], "data:image/png;base64,") {
		t.Fatalf("image inputs = %#v", adapter.optimizeRequest.ImageDataURLs)
	}
}

func TestPlanModelUpdateTreatsEquivalentCapabilitiesAsNoOp(t *testing.T) {
	current := model.AIModel{
		DisplayName:  "GPT Image",
		ModelID:      "gpt-image-2",
		Type:         "image",
		Capabilities: datatypes.JSON(`{"imageToImage":true,"textToImage":true,"promptOptimization":false}`),
		Enabled:      true,
		TestStatus:   StatusHealthy,
	}
	displayName := "  GPT Image  "
	modelID := " gpt-image-2 "
	enabled := true
	capabilities := Capabilities{TextToImage: true, ImageToImage: true}

	plan, err := planModelUpdate(current, UpdateModelInput{
		DisplayName:  &displayName,
		ModelID:      &modelID,
		Enabled:      &enabled,
		Capabilities: &capabilities,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.updates) != 0 {
		t.Fatalf("updates = %#v", plan.updates)
	}
	if plan.invalidateTest {
		t.Fatal("equivalent capabilities must not invalidate a healthy test")
	}
}

func TestPlanModelUpdateOnlyInvalidatesTestForRuntimeConfiguration(t *testing.T) {
	current := model.AIModel{
		DisplayName:  "GPT Image",
		ModelID:      "gpt-image-2",
		Type:         "image",
		Capabilities: datatypes.JSON(`{"textToImage":true,"imageToImage":true}`),
		Enabled:      true,
		TestStatus:   StatusHealthy,
	}

	newName := "GPT Image 2"
	disabled := false
	plan, err := planModelUpdate(current, UpdateModelInput{
		DisplayName: &newName,
		Enabled:     &disabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.invalidateTest {
		t.Fatal("display name and enabled changes must not invalidate the capability test")
	}
	if !plan.disabling {
		t.Fatal("enabled to disabled transition must require a binding check")
	}
	if len(plan.updates) != 2 {
		t.Fatalf("updates = %#v", plan.updates)
	}

	newModelID := "gpt-image-3"
	plan, err = planModelUpdate(current, UpdateModelInput{ModelID: &newModelID})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.invalidateTest {
		t.Fatal("model ID change must invalidate the capability test")
	}

	capabilities := Capabilities{TextToImage: true}
	plan, err = planModelUpdate(current, UpdateModelInput{Capabilities: &capabilities})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.invalidateTest {
		t.Fatal("capability change must invalidate the capability test")
	}
}

type recordingAdapter struct {
	optimizeRequest provider.OptimizePromptRequest
}

func (a *recordingAdapter) OptimizePrompt(
	_ context.Context,
	request provider.OptimizePromptRequest,
) (provider.PromptResult, error) {
	a.optimizeRequest = request
	return provider.PromptResult{Content: "OK"}, nil
}

func (a *recordingAdapter) GenerateTextToImage(
	context.Context,
	provider.TextToImageRequest,
) ([]provider.GeneratedImage, error) {
	return nil, nil
}

func (a *recordingAdapter) GenerateImageToImage(
	context.Context,
	provider.ImageToImageRequest,
) ([]provider.GeneratedImage, error) {
	return nil, nil
}

func (a *recordingAdapter) TestConnection(
	context.Context,
) (provider.ConnectionTestResult, error) {
	return provider.ConnectionTestResult{}, nil
}

func (a *recordingAdapter) TestModel(
	context.Context,
	provider.ModelTestRequest,
) (provider.ModelTestResult, error) {
	return provider.ModelTestResult{}, nil
}
