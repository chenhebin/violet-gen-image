package provider

import (
	"context"
	"io"
	"time"
)

const (
	ModelTypeChat  ModelType = "chat"
	ModelTypeImage ModelType = "image"

	ImageSourceBase64 ImageSource = "base64"
	ImageSourceURL    ImageSource = "url"
)

type ModelType string

type ImageSource string

// Adapter is the boundary used by prompt and generation services. Provider
// credentials and protocol-specific response shapes stay behind this interface.
type Adapter interface {
	OptimizePrompt(context.Context, OptimizePromptRequest) (PromptResult, error)
	GenerateTextToImage(context.Context, TextToImageRequest) ([]GeneratedImage, error)
	GenerateImageToImage(context.Context, ImageToImageRequest) ([]GeneratedImage, error)
	TestConnection(context.Context) (ConnectionTestResult, error)
	TestModel(context.Context, ModelTestRequest) (ModelTestResult, error)
}

type OptimizePromptRequest struct {
	Model         string
	SystemPrompt  string
	Prompt        string
	ImageDataURLs []string
	Temperature   *float64
	MaxTokens     int
}

type PromptResult struct {
	Content        string
	Model          string
	RequestID      string
	Usage          Usage
	RequestSummary CallMetadata
}

// CallMetadata is a bounded, secret-free description of one provider call.
// ParameterSummary must contain only counts, lengths, and configuration flags.
type CallMetadata struct {
	Operation        string         `json:"operation"`
	Method           string         `json:"method"`
	Path             string         `json:"path"`
	Model            string         `json:"model,omitempty"`
	ParameterSummary map[string]any `json:"parameterSummary,omitempty"`
	Status           int            `json:"status,omitempty"`
	LatencyMillis    int64          `json:"latencyMs"`
	RequestID        string         `json:"requestId,omitempty"`
	ErrorKind        string         `json:"errorKind,omitempty"`
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type TextToImageRequest struct {
	Model         string
	Prompt        string
	OutputCount   int
	Size          string
	Quality       string
	Style         string
	UserReference string
}

// ImageInput is consumed once by GenerateImageToImage. Callers should reopen
// files when issuing a new provider request.
type ImageInput struct {
	Filename    string
	ContentType string
	Reader      io.Reader
}

type ImageToImageRequest struct {
	Model         string
	Prompt        string
	Images        []ImageInput
	OutputCount   int
	Size          string
	Quality       string
	UserReference string
}

type GeneratedImage struct {
	Data           []byte
	ContentType    string
	RevisedPrompt  string
	Source         ImageSource
	RequestID      string
	RequestSummary CallMetadata
}

type ConnectionTestResult struct {
	Latency        time.Duration
	RequestID      string
	ModelCount     int
	RequestSummary CallMetadata
}

type ModelTestRequest struct {
	Model  string
	Type   ModelType
	Prompt string
	Image  *ImageInput
}

type ModelTestResult struct {
	Latency        time.Duration
	RequestID      string
	Model          string
	RequestSummary CallMetadata
}
