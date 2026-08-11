package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/ai"
	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/provider"
)

const (
	StatusDraft     = "draft"
	StatusConfirmed = "confirmed"
)

type Sections struct {
	Subject     string `json:"subject"`
	Scene       string `json:"scene"`
	Style       string `json:"style"`
	Composition string `json:"composition"`
	Details     string `json:"details"`
	Negative    string `json:"negative"`
	Output      string `json:"output"`
}

type ReferenceAsset struct {
	AssetID string `json:"assetId"`
	Role    string `json:"role"`
}

type OptimizeInput struct {
	Source          string           `json:"source"`
	Mode            string           `json:"mode"`
	SourceAssetIDs  []string         `json:"sourceAssetIds"`
	ReferenceAssets []ReferenceAsset `json:"referenceAssets"`
}

type ConfirmInput struct {
	ID       string   `json:"id"`
	Source   string   `json:"source"`
	Sections Sections `json:"sections"`
}

type DTO struct {
	ID          string     `json:"id"`
	Source      string     `json:"source"`
	Sections    Sections   `json:"sections"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
}

type Service struct {
	db      *gorm.DB
	assets  *asset.Service
	factory *ai.Factory
	logger  *slog.Logger
}

func New(db *gorm.DB, assets *asset.Service, factory *ai.Factory, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, assets: assets, factory: factory, logger: logger}
}

func (s *Service) Optimize(ctx context.Context, userID string, input OptimizeInput) (*DTO, error) {
	input.Source = strings.TrimSpace(input.Source)
	if len([]rune(input.Source)) < 4 || len([]rune(input.Source)) > 2000 {
		return nil, apierror.Invalid("需求描述长度需为 4 到 2000 字", nil)
	}
	imageCount := len(input.SourceAssetIDs) + len(input.ReferenceAssets)
	if imageCount > 8 {
		return nil, apierror.Invalid("参与优化的素材不能超过 8 张", nil)
	}
	input.Mode = promptModeForImageCount(imageCount)

	providerModel, aiModel, err := s.currentModel(ctx, "chat")
	if err != nil {
		s.logger.WarnContext(ctx, "prompt_optimization_model_unavailable",
			"binding_type", "chat",
			"error", err,
		)
		return nil, err
	}
	capabilities := map[string]bool{}
	if err := json.Unmarshal(aiModel.Capabilities, &capabilities); err != nil {
		return nil, apierror.Internal(err)
	}
	if !capabilities["promptOptimization"] {
		return nil, apierror.New(http.StatusConflict, apierror.CodeAICapability, "当前对话模型不支持提示词优化", nil)
	}

	imageDataURLs := make([]string, 0, len(input.SourceAssetIDs)+len(input.ReferenceAssets))
	for _, assetID := range input.SourceAssetIDs {
		assetModel, err := s.assets.GetOwned(ctx, userID, assetID)
		if err != nil {
			return nil, err
		}
		if assetModel.Kind != asset.KindSource {
			return nil, apierror.Invalid("原图素材用途不匹配", map[string]string{"assetId": assetID})
		}
		dataURL, err := s.assets.DataURL(ctx, *assetModel)
		if err != nil {
			return nil, err
		}
		imageDataURLs = append(imageDataURLs, dataURL)
	}
	for _, reference := range input.ReferenceAssets {
		assetModel, err := s.assets.GetOwned(ctx, userID, reference.AssetID)
		if err != nil {
			return nil, err
		}
		if assetModel.Kind != asset.KindReference || !IsValidReferenceRole(reference.Role) {
			return nil, apierror.Invalid("参考图素材或用途无效", map[string]string{"assetId": reference.AssetID})
		}
		dataURL, err := s.assets.DataURL(ctx, *assetModel)
		if err != nil {
			return nil, err
		}
		imageDataURLs = append(imageDataURLs, dataURL)
	}
	if len(imageDataURLs) > 0 && !capabilities["visionInput"] {
		return nil, apierror.New(http.StatusConflict, apierror.CodeAICapability, "当前对话模型不支持图片输入", nil)
	}

	adapter, err := s.factory.FromProvider(providerModel)
	if err != nil {
		s.logger.ErrorContext(ctx, "prompt_optimization_provider_invalid",
			"provider_code", providerModel.Code,
			"provider_id", providerModel.ID,
			"model", aiModel.ModelID,
			"error", err,
		)
		return nil, apierror.New(http.StatusBadGateway, apierror.CodeAIProvider, "AI 服务配置不可用", nil)
	}
	startedAt := time.Now()
	s.logger.InfoContext(ctx, "prompt_optimization_provider_call_started",
		"provider_code", providerModel.Code,
		"provider_id", providerModel.ID,
		"model", aiModel.ModelID,
		"mode", input.Mode,
		"image_count", len(imageDataURLs),
	)
	result, err := adapter.OptimizePrompt(ctx, provider.OptimizePromptRequest{
		Model:         aiModel.ModelID,
		SystemPrompt:  optimizationSystemPrompt,
		Prompt:        buildOptimizationInput(input),
		ImageDataURLs: imageDataURLs,
		MaxTokens:     1800,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "prompt_optimization_provider_call_failed",
			"provider_code", providerModel.Code,
			"provider_id", providerModel.ID,
			"model", aiModel.ModelID,
			"latency_ms", time.Since(startedAt).Milliseconds(),
			"error", err,
		)
		return nil, apierror.New(http.StatusBadGateway, apierror.CodeAIProvider, "提示词优化服务暂时不可用", nil)
	}
	s.logger.InfoContext(ctx, "prompt_optimization_provider_call_succeeded",
		"provider_code", providerModel.Code,
		"provider_id", providerModel.ID,
		"model", aiModel.ModelID,
		"upstream_request_id", result.RequestID,
		"prompt_tokens", result.Usage.PromptTokens,
		"completion_tokens", result.Usage.CompletionTokens,
		"total_tokens", result.Usage.TotalTokens,
		"latency_ms", time.Since(startedAt).Milliseconds(),
	)
	sections, err := parseSections(result.Content)
	if err != nil {
		return nil, apierror.New(
			http.StatusBadGateway,
			apierror.CodeAIProvider,
			"AI 返回的提示词结构无效，请重试",
			nil,
		)
	}

	sourceIDs, _ := json.Marshal(input.SourceAssetIDs)
	references, _ := json.Marshal(input.ReferenceAssets)
	sectionJSON, _ := json.Marshal(sections)
	version := model.PromptVersion{
		UserID:                userID,
		Source:                input.Source,
		Mode:                  input.Mode,
		Sections:              datatypes.JSON(sectionJSON),
		SourceAssetIDs:        datatypes.JSON(sourceIDs),
		ReferenceAssets:       datatypes.JSON(references),
		ProviderID:            &providerModel.ID,
		ModelID:               &aiModel.ID,
		ProviderConfigVersion: providerModel.ConfigVersion,
		ModelConfigVersion:    aiModel.ConfigVersion,
		Status:                StatusDraft,
	}
	if err := s.db.WithContext(ctx).Create(&version).Error; err != nil {
		return nil, err
	}
	return dto(version, sections), nil
}

func (s *Service) Confirm(ctx context.Context, userID string, input ConfirmInput) (*DTO, error) {
	input.Source = strings.TrimSpace(input.Source)
	if input.ID == "" || input.Source == "" {
		return nil, apierror.Invalid("提示词版本和原始需求不能为空", nil)
	}
	if err := validateSections(input.Sections); err != nil {
		return nil, err
	}

	var result model.PromptVersion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var draft model.PromptVersion
		err := tx.Where("id = ? AND user_id = ?", input.ID, userID).First(&draft).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierror.New(http.StatusNotFound, apierror.CodePromptNotFound, "提示词版本不存在", nil)
		}
		if err != nil {
			return err
		}
		sectionJSON, _ := json.Marshal(input.Sections)
		now := time.Now().UTC()
		result = model.PromptVersion{
			UserID:                userID,
			Source:                input.Source,
			Mode:                  draft.Mode,
			Sections:              datatypes.JSON(sectionJSON),
			SourceAssetIDs:        draft.SourceAssetIDs,
			ReferenceAssets:       draft.ReferenceAssets,
			ProviderID:            draft.ProviderID,
			ModelID:               draft.ModelID,
			ProviderConfigVersion: draft.ProviderConfigVersion,
			ModelConfigVersion:    draft.ModelConfigVersion,
			Status:                StatusConfirmed,
			ConfirmedAt:           &now,
		}
		return tx.Create(&result).Error
	})
	if err != nil {
		return nil, err
	}
	return dto(result, input.Sections), nil
}

func (s *Service) GetConfirmed(ctx context.Context, userID string, promptID string) (*model.PromptVersion, Sections, error) {
	var version model.PromptVersion
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND status = ?", promptID, userID, StatusConfirmed).
		First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, Sections{}, apierror.New(http.StatusNotFound, apierror.CodePromptNotFound, "已确认提示词版本不存在", nil)
	}
	if err != nil {
		return nil, Sections{}, err
	}
	var sections Sections
	if err := json.Unmarshal(version.Sections, &sections); err != nil {
		return nil, Sections{}, apierror.Internal(err)
	}
	return &version, sections, nil
}

func BuildGenerationPrompt(source string, sections Sections) string {
	values := []string{"用户需求：" + strings.TrimSpace(source)}
	sectionValues := []struct {
		label string
		value string
	}{
		{label: "主体", value: sections.Subject},
		{label: "场景", value: sections.Scene},
		{label: "风格", value: sections.Style},
		{label: "构图", value: sections.Composition},
		{label: "细节", value: sections.Details},
		{label: "避免", value: sections.Negative},
		{label: "输出要求", value: sections.Output},
	}
	for _, section := range sectionValues {
		if value := strings.TrimSpace(section.value); value != "" {
			values = append(values, section.label+"："+value)
		}
	}
	return strings.Join(values, "\n")
}

func (s *Service) currentModel(ctx context.Context, bindingType string) (model.AIProvider, model.AIModel, error) {
	var binding model.PlatformModelBinding
	if err := s.db.WithContext(ctx).Where("binding_type = ?", bindingType).First(&binding).Error; err != nil {
		return model.AIProvider{}, model.AIModel{}, apierror.New(
			http.StatusServiceUnavailable,
			apierror.CodeAIUnavailable,
			"平台 AI 能力尚未配置",
			nil,
		)
	}
	var providerModel model.AIProvider
	var aiModel model.AIModel
	if err := s.db.WithContext(ctx).First(&providerModel, "id = ?", binding.ProviderID).Error; err != nil {
		return model.AIProvider{}, model.AIModel{}, apierror.New(http.StatusServiceUnavailable, apierror.CodeAIUnavailable, "AI 服务商不可用", nil)
	}
	if err := s.db.WithContext(ctx).First(&aiModel, "id = ?", binding.ModelID).Error; err != nil {
		return model.AIProvider{}, model.AIModel{}, apierror.New(http.StatusServiceUnavailable, apierror.CodeAIUnavailable, "AI 模型不可用", nil)
	}
	if !providerModel.Enabled || providerModel.ConnectionStatus != "healthy" ||
		!aiModel.Enabled || aiModel.TestStatus != "healthy" ||
		providerModel.ConfigVersion != binding.ProviderConfigVersion ||
		aiModel.ConfigVersion != binding.ModelConfigVersion {
		return model.AIProvider{}, model.AIModel{}, apierror.New(
			http.StatusServiceUnavailable,
			apierror.CodeAIUnavailable,
			"平台 AI 模型需要重新测试",
			nil,
		)
	}
	return providerModel, aiModel, nil
}

func parseSections(content string) (Sections, error) {
	content = strings.TrimSpace(content)
	if start := strings.Index(content, "{"); start >= 0 {
		if end := strings.LastIndex(content, "}"); end >= start {
			content = content[start : end+1]
		}
	}
	var sections Sections
	if err := json.Unmarshal([]byte(content), &sections); err != nil {
		return Sections{}, err
	}
	if err := validateSections(sections); err != nil {
		return Sections{}, err
	}
	return sections, nil
}

func validateSections(sections Sections) error {
	fields := []string{
		sections.Subject,
		sections.Scene,
		sections.Style,
		sections.Composition,
		sections.Details,
		sections.Negative,
		sections.Output,
	}
	total := 0
	for _, value := range fields {
		if strings.TrimSpace(value) == "" {
			return apierror.Invalid("提示词的七个分区都不能为空", nil)
		}
		total += len([]rune(value))
	}
	if total > 8000 {
		return apierror.Invalid("优化提示词内容过长", nil)
	}
	return nil
}

func IsValidReferenceRole(role string) bool {
	switch role {
	case "style", "composition", "person", "detail":
		return true
	default:
		return false
	}
}

func promptModeForImageCount(imageCount int) string {
	if imageCount > 0 {
		return "image-to-image"
	}
	return "text-to-image"
}

func buildOptimizationInput(input OptimizeInput) string {
	roles := make([]string, 0, len(input.ReferenceAssets))
	for _, reference := range input.ReferenceAssets {
		roles = append(roles, reference.Role)
	}
	return fmt.Sprintf(
		"创作模式：%s\n用户原始需求：%s\n原图数量：%d\n参考图用途：%s",
		input.Mode,
		input.Source,
		len(input.SourceAssetIDs),
		strings.Join(roles, "、"),
	)
}

func dto(version model.PromptVersion, sections Sections) *DTO {
	return &DTO{
		ID:          version.ID,
		Source:      version.Source,
		Sections:    sections,
		ConfirmedAt: version.ConfirmedAt,
	}
}

const optimizationSystemPrompt = `你是专业商业修图提示词编辑。请结合用户文字和图片，将需求整理成可执行的中文生图提示词。
只返回一个 JSON 对象，不要使用 Markdown。必须恰好包含以下字符串字段：
subject、scene、style、composition、details、negative、output。
保留人物身份与用户明确要求，不编造敏感属性。negative 用于描述需要避免的内容。`
