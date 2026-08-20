package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRequestTimeout        = 180 * time.Second
	defaultResponseHeaderTimeout = 180 * time.Second
	defaultMaxRequestBytes       = 192 << 20
	defaultMaxResponseBytes      = 192 << 20
	defaultMaxImageBytes         = 30 << 20
)

type Config struct {
	BaseURL               string
	APIKey                string
	RequestTimeout        time.Duration
	ConnectTimeout        time.Duration
	ResponseHeaderTimeout time.Duration
	MaxRequestBytes       int64
	MaxResponseBytes      int64
	MaxImageBytes         int64
	MaxRedirects          int
	AllowHTTP             bool
	AllowPrivateNetwork   bool
	Resolver              Resolver
	DialContext           DialContextFunc
}

type OpenAICompatible struct {
	baseURL          *url.URL
	apiKey           string
	client           *http.Client
	policy           OutboundPolicy
	maxRequestBytes  int64
	maxResponseBytes int64
	maxImageBytes    int64
}

var _ Adapter = (*OpenAICompatible)(nil)

func NewOpenAICompatible(cfg Config) (*OpenAICompatible, error) {
	policy := OutboundPolicy{
		AllowHTTP:           cfg.AllowHTTP,
		AllowPrivateNetwork: cfg.AllowPrivateNetwork,
		Resolver:            cfg.Resolver,
		DialContext:         cfg.DialContext,
		ConnectTimeout:      cfg.ConnectTimeout,
	}
	baseURL, err := parseOutboundURL(cfg.BaseURL, policy)
	if err != nil {
		return nil, newError(ErrorInvalidConfig, "configure", err)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, newError(ErrorInvalidConfig, "configure", errors.New("API key is required"))
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	if cfg.ResponseHeaderTimeout <= 0 {
		cfg.ResponseHeaderTimeout = defaultResponseHeaderTimeout
	}
	if cfg.MaxRequestBytes <= 0 {
		cfg.MaxRequestBytes = defaultMaxRequestBytes
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = defaultMaxResponseBytes
	}
	if cfg.MaxImageBytes <= 0 {
		cfg.MaxImageBytes = defaultMaxImageBytes
	}

	return &OpenAICompatible{
		baseURL:          baseURL,
		apiKey:           cfg.APIKey,
		client:           newSafeHTTPClient(policy, cfg.ResponseHeaderTimeout, cfg.RequestTimeout, cfg.MaxRedirects),
		policy:           policy,
		maxRequestBytes:  cfg.MaxRequestBytes,
		maxResponseBytes: cfg.MaxResponseBytes,
		maxImageBytes:    cfg.MaxImageBytes,
	}, nil
}

func (a *OpenAICompatible) OptimizePrompt(ctx context.Context, request OptimizePromptRequest) (PromptResult, error) {
	if strings.TrimSpace(request.Model) == "" || strings.TrimSpace(request.Prompt) == "" {
		return PromptResult{}, newError(ErrorInvalidRequest, "optimize_prompt", errors.New("model and prompt are required"))
	}
	if request.MaxTokens < 0 {
		return PromptResult{}, newError(ErrorInvalidRequest, "optimize_prompt", errors.New("max tokens must not be negative"))
	}

	userContent := any(request.Prompt)
	if len(request.ImageDataURLs) > 0 {
		content := make([]chatContent, 0, len(request.ImageDataURLs)+1)
		content = append(content, chatContent{Type: "text", Text: request.Prompt})
		for _, imageURL := range request.ImageDataURLs {
			if err := validateImageDataURL(imageURL, a.maxImageBytes); err != nil {
				return PromptResult{}, newError(ErrorInvalidRequest, "optimize_prompt", err)
			}
			content = append(content, chatContent{
				Type: "image_url",
				ImageURL: &chatImageURL{
					URL:    imageURL,
					Detail: "auto",
				},
			})
		}
		userContent = content
	}

	messages := make([]chatMessage, 0, 2)
	if strings.TrimSpace(request.SystemPrompt) != "" {
		messages = append(messages, chatMessage{Role: "system", Content: request.SystemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: userContent})
	payload := chatRequest{
		Model:       request.Model,
		Messages:    messages,
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
		Stream:      false,
	}
	var response chatResponse
	metadata, err := a.doJSON(ctx, "optimize_prompt", http.MethodPost, "/v1/chat/completions", payload, &response)
	metadata.Model = request.Model
	metadata.ParameterSummary = map[string]any{
		"promptLength":           len([]rune(request.Prompt)),
		"imageCount":             len(request.ImageDataURLs),
		"systemPromptConfigured": strings.TrimSpace(request.SystemPrompt) != "",
		"temperatureConfigured":  request.Temperature != nil,
		"maxTokensConfigured":    request.MaxTokens > 0,
	}
	if err != nil {
		attachMetadata(err, metadata)
		return PromptResult{}, err
	}
	if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return PromptResult{}, withMetadata(
			newError(ErrorInvalidResponse, "optimize_prompt", errors.New("provider returned no message")),
			metadata,
		)
	}
	return PromptResult{
		Content:        response.Choices[0].Message.Content,
		Model:          response.Model,
		RequestID:      metadata.RequestID,
		RequestSummary: metadata,
		Usage: Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}, nil
}

func (a *OpenAICompatible) GenerateTextToImage(ctx context.Context, request TextToImageRequest) ([]GeneratedImage, error) {
	if err := validateImageRequest(request.Model, request.Prompt, request.OutputCount); err != nil {
		return nil, newError(ErrorInvalidRequest, "generate_image", err)
	}
	payload := imageGenerationRequest{
		Model:   request.Model,
		Prompt:  request.Prompt,
		N:       optionalOutputCount(request.OutputCount),
		Size:    request.Size,
		Quality: request.Quality,
		Style:   request.Style,
	}
	var response imageResponse
	metadata, err := a.doJSON(ctx, "generate_image", http.MethodPost, "/v1/images/generations", payload, &response)
	metadata.Model = request.Model
	metadata.ParameterSummary = map[string]any{
		"promptLength":          len([]rune(request.Prompt)),
		"outputCount":           request.OutputCount,
		"multiOutputConfigured": request.OutputCount > 1,
		"sizeConfigured":        strings.TrimSpace(request.Size) != "",
		"qualityConfigured":     strings.TrimSpace(request.Quality) != "",
		"styleConfigured":       strings.TrimSpace(request.Style) != "",
	}
	if err != nil {
		attachMetadata(err, metadata)
		return nil, err
	}
	return a.parseImages(ctx, "generate_image", metadata, response)
}

func (a *OpenAICompatible) GenerateImageToImage(ctx context.Context, request ImageToImageRequest) ([]GeneratedImage, error) {
	if err := validateImageRequest(request.Model, request.Prompt, request.OutputCount); err != nil {
		return nil, newError(ErrorInvalidRequest, "edit_image", err)
	}
	if len(request.Images) == 0 {
		return nil, newError(ErrorInvalidRequest, "edit_image", errors.New("at least one input image is required"))
	}
	if len(request.Images) > 8 {
		return nil, newError(ErrorInvalidRequest, "edit_image", errors.New("at most eight input images are allowed"))
	}
	for index, image := range request.Images {
		if image.Reader == nil {
			return nil, newError(
				ErrorInvalidRequest,
				"edit_image",
				fmt.Errorf("image %d has no reader", index+1),
			)
		}
		if err := validateInputImageContentType(image.ContentType); err != nil {
			return nil, newError(ErrorInvalidRequest, "edit_image", err)
		}
	}
	if int64(len(request.Prompt)) > a.maxRequestBytes {
		return nil, newError(ErrorInvalidRequest, "edit_image", errors.New("prompt is too large"))
	}

	bodyReader, contentType, writerErrors := streamMultipart(
		request,
		a.maxImageBytes,
		a.maxRequestBytes,
	)
	var response imageResponse
	metadata, err := a.doStream(ctx, "edit_image", "/v1/images/edits", bodyReader, contentType, &response)
	metadata.Model = request.Model
	metadata.ParameterSummary = map[string]any{
		"promptLength":      len([]rune(request.Prompt)),
		"imageCount":        len(request.Images),
		"outputCount":       request.OutputCount,
		"sizeConfigured":    strings.TrimSpace(request.Size) != "",
		"qualityConfigured": strings.TrimSpace(request.Quality) != "",
	}
	writerErr := <-writerErrors
	if err != nil {
		attachMetadata(err, metadata)
		return nil, err
	}
	if writerErr != nil {
		return nil, newError(ErrorInvalidRequest, "edit_image", writerErr)
	}
	return a.parseImages(ctx, "edit_image", metadata, response)
}

func (a *OpenAICompatible) TestConnection(ctx context.Context) (ConnectionTestResult, error) {
	startedAt := time.Now()
	var response modelsResponse
	metadata, err := a.doJSON(ctx, "test_connection", http.MethodGet, "/v1/models", nil, &response)
	if err != nil {
		attachMetadata(err, metadata)
		return ConnectionTestResult{}, err
	}
	return ConnectionTestResult{
		Latency:        time.Since(startedAt),
		RequestID:      metadata.RequestID,
		ModelCount:     len(response.Data),
		RequestSummary: metadata,
	}, nil
}

func (a *OpenAICompatible) TestModel(ctx context.Context, request ModelTestRequest) (ModelTestResult, error) {
	if strings.TrimSpace(request.Model) == "" {
		return ModelTestResult{}, newError(ErrorInvalidRequest, "test_model", errors.New("model is required"))
	}
	prompt := strings.TrimSpace(request.Prompt)
	startedAt := time.Now()
	result := ModelTestResult{Model: request.Model}

	switch request.Type {
	case ModelTypeChat:
		if prompt == "" {
			prompt = "Reply with OK."
		}
		response, err := a.OptimizePrompt(ctx, OptimizePromptRequest{
			Model:  request.Model,
			Prompt: prompt,
		})
		if err != nil {
			return ModelTestResult{}, err
		}
		result.RequestID = response.RequestID
		result.RequestSummary = response.RequestSummary
	case ModelTypeImage:
		if prompt == "" {
			prompt = "A simple black square centered on a white background."
		}
		if request.Image == nil {
			images, err := a.GenerateTextToImage(ctx, TextToImageRequest{
				Model:       request.Model,
				Prompt:      prompt,
				OutputCount: 1,
			})
			if err != nil {
				return ModelTestResult{}, err
			}
			result.RequestID = firstRequestID(images)
			if len(images) > 0 {
				result.RequestSummary = images[0].RequestSummary
			}
		} else {
			images, err := a.GenerateImageToImage(ctx, ImageToImageRequest{
				Model:       request.Model,
				Prompt:      prompt,
				Images:      []ImageInput{*request.Image},
				OutputCount: 1,
			})
			if err != nil {
				return ModelTestResult{}, err
			}
			result.RequestID = firstRequestID(images)
			if len(images) > 0 {
				result.RequestSummary = images[0].RequestSummary
			}
		}
	default:
		return ModelTestResult{}, newError(ErrorInvalidRequest, "test_model", errors.New("unsupported model type"))
	}
	result.Latency = time.Since(startedAt)
	return result, nil
}

func (a *OpenAICompatible) doJSON(
	ctx context.Context,
	operation string,
	method string,
	endpoint string,
	payload any,
	output any,
) (CallMetadata, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return CallMetadata{Operation: operation, Method: method, Path: endpoint}, newError(ErrorInvalidRequest, operation, err)
		}
		if int64(len(encoded)) > a.maxRequestBytes {
			return CallMetadata{Operation: operation, Method: method, Path: endpoint}, newError(ErrorInvalidRequest, operation, errors.New("request body is too large"))
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, a.endpoint(endpoint), body)
	if err != nil {
		return CallMetadata{Operation: operation, Method: method, Path: endpoint}, newError(ErrorInvalidRequest, operation, err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return a.execute(operation, request, output)
}

func (a *OpenAICompatible) doStream(
	ctx context.Context,
	operation string,
	endpoint string,
	body io.Reader,
	contentType string,
	output any,
) (CallMetadata, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint(endpoint), body)
	if err != nil {
		return CallMetadata{Operation: operation, Method: http.MethodPost, Path: endpoint}, newError(ErrorInvalidRequest, operation, err)
	}
	defer request.Body.Close()
	request.Header.Set("Content-Type", contentType)
	return a.execute(operation, request, output)
}

func (a *OpenAICompatible) execute(operation string, request *http.Request, output any) (CallMetadata, error) {
	startedAt := time.Now()
	metadata := CallMetadata{Operation: operation, Method: request.Method, Path: request.URL.Path}
	finish := func(err error) (CallMetadata, error) {
		metadata.LatencyMillis = time.Since(startedAt).Milliseconds()
		if err != nil {
			attachMetadata(err, metadata)
		}
		return metadata, err
	}
	if err := validateResolvedTarget(request.Context(), request.URL, a.policy); err != nil {
		return finish(newError(ErrorUnsafeURL, operation, err))
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+a.apiKey)
	request.Header.Set("User-Agent", "yingyan-backend/0.1")

	response, err := a.client.Do(request)
	if err != nil {
		return finish(classifyTransportError(operation, err))
	}
	defer response.Body.Close()

	requestID := response.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = response.Header.Get("Request-Id")
	}
	metadata.RequestID = requestID
	metadata.Status = response.StatusCode
	body, err := readLimited(response.Body, a.maxResponseBytes)
	if err != nil {
		return finish(newError(ErrorResponseTooLarge, operation, err))
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return finish(parseProviderHTTPError(operation, response.StatusCode, body))
	}
	if output == nil {
		return finish(nil)
	}
	if err := json.Unmarshal(body, output); err != nil {
		return finish(newError(ErrorInvalidResponse, operation, err))
	}
	return finish(nil)
}

func (a *OpenAICompatible) parseImages(
	ctx context.Context,
	operation string,
	metadata CallMetadata,
	response imageResponse,
) ([]GeneratedImage, error) {
	if len(response.Data) == 0 {
		return nil, newError(ErrorInvalidResponse, operation, errors.New("provider returned no images"))
	}
	images := make([]GeneratedImage, 0, len(response.Data))
	for _, item := range response.Data {
		switch {
		case strings.TrimSpace(item.Base64JSON) != "":
			data, contentType, err := decodeImage(item.Base64JSON, a.maxImageBytes)
			if err != nil {
				return nil, withMetadata(newError(ErrorInvalidResponse, operation, err), metadata)
			}
			images = append(images, GeneratedImage{
				Data:           data,
				ContentType:    contentType,
				RevisedPrompt:  item.RevisedPrompt,
				Source:         ImageSourceBase64,
				RequestID:      metadata.RequestID,
				RequestSummary: metadata,
			})
		case strings.TrimSpace(item.URL) != "":
			data, contentType, err := a.fetchImage(ctx, item.URL)
			if err != nil {
				return nil, withMetadata(err, metadata)
			}
			images = append(images, GeneratedImage{
				Data:           data,
				ContentType:    contentType,
				RevisedPrompt:  item.RevisedPrompt,
				Source:         ImageSourceURL,
				RequestID:      metadata.RequestID,
				RequestSummary: metadata,
			})
		default:
			return nil, withMetadata(
				newError(ErrorInvalidResponse, operation, errors.New("provider image has no data")),
				metadata,
			)
		}
	}
	return images, nil
}

func (a *OpenAICompatible) fetchImage(ctx context.Context, rawURL string) ([]byte, string, error) {
	target, err := parseResourceURL(rawURL, a.policy)
	if err != nil {
		return nil, "", newError(ErrorUnsafeURL, "fetch_image", err)
	}
	if err := validateResolvedTarget(ctx, target, a.policy); err != nil {
		return nil, "", newError(ErrorUnsafeURL, "fetch_image", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, "", newError(ErrorInvalidResponse, "fetch_image", err)
	}
	request.Header.Set("Accept", "image/png,image/jpeg,image/webp")
	response, err := a.client.Do(request)
	if err != nil {
		return nil, "", classifyTransportError("fetch_image", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", &Error{
			Kind:       ErrorRejected,
			Operation:  "fetch_image",
			StatusCode: response.StatusCode,
			Retryable:  response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500,
		}
	}
	data, err := readLimited(response.Body, a.maxImageBytes)
	if err != nil {
		return nil, "", newError(ErrorResponseTooLarge, "fetch_image", err)
	}
	contentType, err := detectImageContentType(data, response.Header.Get("Content-Type"))
	if err != nil {
		return nil, "", newError(ErrorInvalidResponse, "fetch_image", err)
	}
	return data, contentType, nil
}

func (a *OpenAICompatible) endpoint(endpointPath string) string {
	result := *a.baseURL
	basePath := strings.TrimRight(result.Path, "/")
	requestPath := "/" + strings.TrimLeft(endpointPath, "/")
	if strings.HasSuffix(basePath, "/v1") && strings.HasPrefix(requestPath, "/v1/") {
		requestPath = strings.TrimPrefix(requestPath, "/v1")
	}
	result.Path = path.Clean(basePath + requestPath)
	result.RawPath = ""
	return result.String()
}

func streamMultipart(
	request ImageToImageRequest,
	maxImageBytes int64,
	maxRequestBytes int64,
) (io.Reader, string, <-chan error) {
	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	done := make(chan error, 1)
	go func() {
		var writeErr error
		defer func() {
			if closeErr := multipartWriter.Close(); writeErr == nil {
				writeErr = closeErr
			}
			_ = writer.CloseWithError(writeErr)
			done <- writeErr
			close(done)
		}()

		fields := map[string]string{
			"model":   request.Model,
			"prompt":  request.Prompt,
			"size":    request.Size,
			"quality": request.Quality,
		}
		if count := optionalOutputCount(request.OutputCount); count > 0 {
			fields["n"] = strconv.Itoa(count)
		}
		for key, value := range fields {
			if strings.TrimSpace(value) == "" {
				continue
			}
			if writeErr = multipartWriter.WriteField(key, value); writeErr != nil {
				return
			}
		}
		var totalImageBytes int64
		for index, image := range request.Images {
			filename := strings.TrimSpace(image.Filename)
			if filename == "" {
				filename = fmt.Sprintf("image-%d", index+1)
			}
			filename = path.Base(strings.ReplaceAll(filename, "\\", "/"))
			header := makeTextprotoFileHeader("image[]", filename, image.ContentType)
			var part io.Writer
			part, writeErr = multipartWriter.CreatePart(header)
			if writeErr != nil {
				return
			}
			var written int64
			written, writeErr = io.Copy(part, io.LimitReader(image.Reader, maxImageBytes+1))
			if writeErr != nil {
				return
			}
			if written > maxImageBytes {
				writeErr = fmt.Errorf("image %d exceeds the configured size limit", index+1)
				return
			}
			totalImageBytes += written
			if totalImageBytes > maxRequestBytes {
				writeErr = errors.New("multipart request exceeds the configured size limit")
				return
			}
		}
	}()
	return reader, multipartWriter.FormDataContentType(), done
}

func validateImageRequest(model, prompt string, count int) error {
	if strings.TrimSpace(model) == "" || strings.TrimSpace(prompt) == "" {
		return errors.New("model and prompt are required")
	}
	if count < 0 || count > 4 {
		return errors.New("output count must be between 1 and 4")
	}
	return nil
}

func validateInputImageContentType(value string) error {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil {
		return errors.New("input image has an invalid content type")
	}
	switch strings.ToLower(mediaType) {
	case "image/jpeg", "image/png", "image/webp":
		return nil
	default:
		return errors.New("input image uses an unsupported content type")
	}
}

func normalizedOutputCount(count int) int {
	if count == 0 {
		return 1
	}
	return count
}

func optionalOutputCount(count int) int {
	if normalizedOutputCount(count) <= 1 {
		return 0
	}
	return count
}

func firstRequestID(images []GeneratedImage) string {
	if len(images) == 0 {
		return ""
	}
	return images[0].RequestID
}

func validateImageDataURL(value string, maxBytes int64) error {
	separator := strings.IndexByte(value, ',')
	if separator < 0 {
		return errors.New("image input must be a base64 data URL")
	}
	metadata := strings.ToLower(value[:separator])
	if !strings.HasPrefix(metadata, "data:image/") || !strings.HasSuffix(metadata, ";base64") {
		return errors.New("image input must be a base64 image data URL")
	}
	mediaType := strings.TrimSuffix(strings.TrimPrefix(metadata, "data:"), ";base64")
	switch mediaType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return errors.New("image data URL uses an unsupported image type")
	}
	if int64(base64.StdEncoding.DecodedLen(len(value)-separator-1)) > maxBytes {
		return errors.New("image data URL is too large")
	}
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(value[separator+1:]))
	if _, err := io.Copy(io.Discard, io.LimitReader(decoder, maxBytes+1)); err != nil {
		return errors.New("image data URL contains invalid base64")
	}
	return nil
}

func decodeImage(value string, maxBytes int64) ([]byte, string, error) {
	if separator := strings.IndexByte(value, ','); strings.HasPrefix(strings.ToLower(value), "data:") && separator >= 0 {
		value = value[separator+1:]
	}
	if int64(base64.StdEncoding.DecodedLen(len(value))) > maxBytes {
		return nil, "", errors.New("provider image is too large")
	}
	data, err := readLimited(base64.NewDecoder(base64.StdEncoding, strings.NewReader(value)), maxBytes)
	if err != nil {
		return nil, "", errors.New("provider returned invalid base64 image")
	}
	contentType, err := detectImageContentType(data, "")
	if err != nil {
		return nil, "", err
	}
	return data, contentType, nil
}

func detectImageContentType(data []byte, declared string) (string, error) {
	if len(data) == 0 {
		return "", errors.New("provider returned an empty image")
	}
	detected := http.DetectContentType(data)
	if mediaType, _, err := mime.ParseMediaType(declared); err == nil && mediaType != "" && mediaType != "application/octet-stream" {
		if mediaType == "image/jpg" {
			mediaType = "image/jpeg"
		}
		if mediaType != detected {
			return "", errors.New("provider image content type does not match its bytes")
		}
	}
	switch detected {
	case "image/jpeg", "image/png", "image/webp":
		return detected, nil
	default:
		return "", errors.New("provider returned an unsupported image type")
	}
}

func readLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, errors.New("invalid response size limit")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("provider response exceeded the configured limit")
	}
	return data, nil
}

func classifyTransportError(operation string, err error) error {
	var networkError net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return &Error{Kind: ErrorUnavailable, Operation: operation, cause: context.Canceled}
	case errors.Is(err, context.DeadlineExceeded):
		return &Error{Kind: ErrorTimeout, Operation: operation, Retryable: true, cause: context.DeadlineExceeded}
	case errors.As(err, &networkError) && networkError.Timeout():
		return &Error{Kind: ErrorTimeout, Operation: operation, Retryable: true, cause: context.DeadlineExceeded}
	default:
		return &Error{Kind: ErrorUnavailable, Operation: operation, Retryable: true, cause: safeCause(err)}
	}
}

func parseProviderHTTPError(operation string, statusCode int, body []byte) error {
	var payload providerErrorResponse
	_ = json.Unmarshal(body, &payload)
	kind := ErrorRejected
	retryable := statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
	if statusCode >= http.StatusInternalServerError {
		kind = ErrorUnavailable
	}
	return &Error{
		Kind:         kind,
		Operation:    operation,
		StatusCode:   statusCode,
		ProviderCode: boundedIdentifier(payload.Error.Code),
		Retryable:    retryable,
	}
}

func boundedIdentifier(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" || len(text) > 80 {
		return ""
	}
	for _, character := range text {
		if !(character == '-' || character == '_' || character == '.' ||
			character >= '0' && character <= '9' ||
			character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z') {
			return ""
		}
	}
	return text
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatContent struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *chatImageURL `json:"image_url,omitempty"`
}

type chatImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type imageGenerationRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	N       int    `json:"n,omitempty"`
	Size    string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
	Style   string `json:"style,omitempty"`
}

type imageResponse struct {
	Data []struct {
		Base64JSON    string `json:"b64_json"`
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
}

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type providerErrorResponse struct {
	Error struct {
		Code any `json:"code"`
	} `json:"error"`
}
