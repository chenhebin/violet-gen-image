package aiconfig

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/ai"
	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/idempotency"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/platform/security"
	"yingyan.local/backend/internal/provider"
)

const (
	StatusUntested = "untested"
	StatusHealthy  = "healthy"
	StatusError    = "error"
)

type Service struct {
	db            *gorm.DB
	factory       *ai.Factory
	encryptionKey string
	allowHTTP     bool
}

type Capabilities struct {
	PromptOptimization bool `json:"promptOptimization,omitempty"`
	VisionInput        bool `json:"visionInput,omitempty"`
	TextToImage        bool `json:"textToImage,omitempty"`
	ImageToImage       bool `json:"imageToImage,omitempty"`
}

type CreateProviderInput struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Enabled bool   `json:"enabled"`
	Note    string `json:"note"`
}

type UpdateProviderInput struct {
	Name    *string `json:"name"`
	BaseURL *string `json:"baseUrl"`
	Enabled *bool   `json:"enabled"`
	Note    *string `json:"note"`
}

type CreateModelInput struct {
	ProviderID   string       `json:"providerId"`
	DisplayName  string       `json:"displayName"`
	ModelID      string       `json:"modelId"`
	Type         string       `json:"type"`
	Enabled      bool         `json:"enabled"`
	Capabilities Capabilities `json:"capabilities"`
}

type UpdateModelInput struct {
	DisplayName  *string       `json:"displayName"`
	ModelID      *string       `json:"modelId"`
	Enabled      *bool         `json:"enabled"`
	Capabilities *Capabilities `json:"capabilities"`
}

type BindInput struct {
	Type    string  `json:"type"`
	ModelID *string `json:"modelId"`
}

type modelUpdatePlan struct {
	updates        map[string]any
	invalidateTest bool
	disabling      bool
}

func New(db *gorm.DB, factory *ai.Factory, encryptionKey string, allowHTTP bool) *Service {
	return &Service{
		db:            db,
		factory:       factory,
		encryptionKey: encryptionKey,
		allowHTTP:     allowHTTP,
	}
}

func (s *Service) ListProviders(ctx context.Context) ([]model.AIProvider, error) {
	var providers []model.AIProvider
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&providers).Error
	return providers, err
}

func (s *Service) CreateProvider(ctx context.Context, input CreateProviderInput) (*model.AIProvider, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	input.APIKey = strings.TrimSpace(input.APIKey)
	input.Note = strings.TrimSpace(input.Note)
	if input.Name == "" || input.APIKey == "" || !providerCodePattern.MatchString(input.Code) {
		return nil, apierror.Invalid("服务商名称、编码或 API Key 无效", nil)
	}
	if err := validateBaseURL(input.BaseURL, s.allowHTTP); err != nil {
		return nil, err
	}
	ciphertext, err := security.Encrypt([]byte(input.APIKey), s.encryptionKey)
	if err != nil {
		return nil, err
	}
	value := model.AIProvider{
		Name:             input.Name,
		Code:             input.Code,
		Protocol:         "openai-compatible",
		BaseURL:          input.BaseURL,
		APIKeyCiphertext: ciphertext,
		APIKeyMask:       security.MaskSecret(input.APIKey),
		Enabled:          input.Enabled,
		ConnectionStatus: StatusUntested,
		ConfigVersion:    1,
		Notes:            input.Note,
		Version:          1,
	}
	if err := s.db.WithContext(ctx).Create(&value).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apierror.Invalid("服务商编码已存在", nil)
		}
		return nil, err
	}
	return &value, nil
}

func (s *Service) UpdateProvider(
	ctx context.Context,
	providerID string,
	input UpdateProviderInput,
) (*model.AIProvider, error) {
	var output model.AIProvider
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&output, "id = ?", providerID).Error; err != nil {
			return notFoundProvider(err)
		}
		updates := map[string]any{"version": gorm.Expr("version + 1")}
		invalidate := false
		if input.Name != nil {
			name := strings.TrimSpace(*input.Name)
			if name == "" {
				return apierror.Invalid("服务商名称不能为空", nil)
			}
			updates["name"] = name
		}
		if input.Note != nil {
			updates["notes"] = strings.TrimSpace(*input.Note)
		}
		if input.BaseURL != nil {
			baseURL := strings.TrimRight(strings.TrimSpace(*input.BaseURL), "/")
			if err := validateBaseURL(baseURL, s.allowHTTP); err != nil {
				return err
			}
			if baseURL != output.BaseURL {
				updates["base_url"] = baseURL
				invalidate = true
			}
		}
		if input.Enabled != nil {
			if !*input.Enabled {
				var bound int64
				if err := tx.Model(&model.PlatformModelBinding{}).
					Where("provider_id = ?", providerID).Count(&bound).Error; err != nil {
					return err
				}
				if bound > 0 {
					return apierror.New(http.StatusConflict, apierror.CodeInvalidInput, "已绑定的平台服务商不能停用", nil)
				}
			}
			updates["enabled"] = *input.Enabled
		}
		if invalidate {
			updates["connection_status"] = StatusUntested
			updates["last_tested_at"] = nil
			updates["last_test_summary"] = ""
			updates["last_test_details"] = nil
			updates["config_version"] = gorm.Expr("config_version + 1")
			if err := tx.Model(&model.AIModel{}).Where("provider_id = ?", providerID).
				Updates(map[string]any{
					"test_status":       StatusUntested,
					"last_tested_at":    nil,
					"last_test_summary": "",
					"last_test_details": nil,
					"config_version":    gorm.Expr("config_version + 1"),
				}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&output).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&output, "id = ?", providerID).Error
	})
	return &output, err
}

func (s *Service) DeleteProvider(ctx context.Context, providerID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var providerModel model.AIProvider
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&providerModel, "id = ?", providerID).Error; err != nil {
			return notFoundProvider(err)
		}

		var modelCount int64
		if err := tx.Model(&model.AIModel{}).
			Where("provider_id = ?", providerID).Count(&modelCount).Error; err != nil {
			return err
		}
		if modelCount > 0 {
			return apierror.New(
				http.StatusConflict,
				apierror.CodeInvalidInput,
				"服务商仍有关联模型，请先删除模型",
				map[string]any{"modelCount": modelCount},
			)
		}

		var bindingCount int64
		if err := tx.Model(&model.PlatformModelBinding{}).
			Where("provider_id = ?", providerID).Count(&bindingCount).Error; err != nil {
			return err
		}
		if bindingCount > 0 {
			return apierror.New(
				http.StatusConflict,
				apierror.CodeInvalidInput,
				"服务商仍被平台使用，请先解除模型绑定",
				nil,
			)
		}

		if err := tx.Where("provider_id = ?", providerID).
			Delete(&model.ModelTestRun{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&providerModel)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "AI 服务商不存在", nil)
		}
		return nil
	})
}

func (s *Service) RotateKey(ctx context.Context, providerID string, rawKey string) (*model.AIProvider, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return nil, apierror.Invalid("API Key 不能为空", nil)
	}
	ciphertext, err := security.Encrypt([]byte(rawKey), s.encryptionKey)
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AIProvider{}).Where("id = ?", providerID).
			Updates(map[string]any{
				"api_key_ciphertext": ciphertext,
				"api_key_mask":       security.MaskSecret(rawKey),
				"connection_status":  StatusUntested,
				"last_tested_at":     nil,
				"last_test_summary":  "",
				"last_test_details":  nil,
				"config_version":     gorm.Expr("config_version + 1"),
				"version":            gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "AI 服务商不存在", nil)
		}
		return tx.Model(&model.AIModel{}).Where("provider_id = ?", providerID).
			Updates(map[string]any{
				"test_status":       StatusUntested,
				"last_tested_at":    nil,
				"last_test_summary": "",
				"last_test_details": nil,
				"config_version":    gorm.Expr("config_version + 1"),
			}).Error
	})
	if err != nil {
		return nil, err
	}
	var output model.AIProvider
	err = s.db.WithContext(ctx).First(&output, "id = ?", providerID).Error
	return &output, err
}

func (s *Service) TestProvider(ctx context.Context, providerID string, adminID string) (*model.AIProvider, error) {
	var value model.AIProvider
	if err := s.db.WithContext(ctx).First(&value, "id = ?", providerID).Error; err != nil {
		return nil, notFoundProvider(err)
	}
	adapter, err := s.factory.FromProvider(value)
	var details any
	if err == nil {
		result, testErr := adapter.TestConnection(ctx)
		err = testErr
		details = result.RequestSummary
		if err != nil {
			details = providerErrorDetails(err)
		}
	} else {
		details = providerErrorDetails(err)
	}
	status := StatusHealthy
	summary := "连接正常"
	if err != nil {
		status = StatusError
		summary = "连接失败：" + safeError(err)
	}
	now := time.Now().UTC()
	if updateErr := s.db.WithContext(ctx).Model(&value).Updates(map[string]any{
		"connection_status": status,
		"last_tested_at":    &now,
		"last_test_summary": summary,
		"last_test_details": marshalTestDetails(details),
	}).Error; updateErr != nil {
		return nil, updateErr
	}
	_ = s.recordTest(ctx, value.ID, nil, "provider", status, summary, value.ConfigVersion, adminID, details)
	value.ConnectionStatus = status
	value.LastTestedAt = &now
	value.LastTestSummary = summary
	value.LastTestDetails = marshalTestDetails(details)
	return &value, nil
}

func (s *Service) ListModels(ctx context.Context, providerID string) ([]model.AIModel, error) {
	query := s.db.WithContext(ctx)
	if providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}
	var models []model.AIModel
	err := query.Order("created_at DESC").Find(&models).Error
	return models, err
}

func (s *Service) CreateModel(ctx context.Context, input CreateModelInput) (*model.AIModel, error) {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.ModelID = strings.TrimSpace(input.ModelID)
	if input.ProviderID == "" || input.DisplayName == "" || input.ModelID == "" {
		return nil, apierror.Invalid("模型服务商、名称和模型 ID 不能为空", nil)
	}
	if err := validateCapabilities(input.Type, input.Capabilities); err != nil {
		return nil, err
	}
	var providerCount int64
	if err := s.db.WithContext(ctx).Model(&model.AIProvider{}).
		Where("id = ?", input.ProviderID).Count(&providerCount).Error; err != nil {
		return nil, err
	}
	if providerCount != 1 {
		return nil, apierror.Invalid("所属服务商不存在", nil)
	}
	capabilities, _ := json.Marshal(input.Capabilities)
	value := model.AIModel{
		ProviderID:    input.ProviderID,
		DisplayName:   input.DisplayName,
		ModelID:       input.ModelID,
		Type:          input.Type,
		Capabilities:  datatypes.JSON(capabilities),
		Enabled:       input.Enabled,
		TestStatus:    StatusUntested,
		ConfigVersion: 1,
		Version:       1,
	}
	if err := s.db.WithContext(ctx).Create(&value).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apierror.Invalid("同一服务商下模型 ID 已存在", nil)
		}
		return nil, err
	}
	return &value, nil
}

func (s *Service) UpdateModel(ctx context.Context, modelID string, input UpdateModelInput) (*model.AIModel, error) {
	var output model.AIModel
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&output, "id = ?", modelID).Error; err != nil {
			return notFoundModel(err)
		}

		plan, err := planModelUpdate(output, input)
		if err != nil {
			return err
		}
		if plan.disabling {
			var bound int64
			if err := tx.Model(&model.PlatformModelBinding{}).
				Where("model_id = ?", modelID).Count(&bound).Error; err != nil {
				return err
			}
			if bound > 0 {
				return apierror.New(http.StatusConflict, apierror.CodeInvalidInput, "已绑定的平台模型不能停用", nil)
			}
		}
		if len(plan.updates) == 0 {
			return nil
		}
		plan.updates["version"] = gorm.Expr("version + 1")
		if plan.invalidateTest {
			plan.updates["test_status"] = StatusUntested
			plan.updates["last_tested_at"] = nil
			plan.updates["last_test_summary"] = ""
			plan.updates["last_test_details"] = nil
			plan.updates["config_version"] = gorm.Expr("config_version + 1")
		}
		if err := tx.Model(&output).Updates(plan.updates).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return apierror.Invalid("同一服务商下模型 ID 已存在", nil)
			}
			return err
		}
		return tx.First(&output, "id = ?", modelID).Error
	})
	return &output, err
}

func planModelUpdate(current model.AIModel, input UpdateModelInput) (modelUpdatePlan, error) {
	plan := modelUpdatePlan{updates: map[string]any{}}
	if input.DisplayName != nil {
		name := strings.TrimSpace(*input.DisplayName)
		if name == "" {
			return plan, apierror.Invalid("模型名称不能为空", nil)
		}
		if name != current.DisplayName {
			plan.updates["display_name"] = name
		}
	}
	if input.ModelID != nil {
		value := strings.TrimSpace(*input.ModelID)
		if value == "" {
			return plan, apierror.Invalid("模型 ID 不能为空", nil)
		}
		if value != current.ModelID {
			plan.updates["model_id"] = value
			plan.invalidateTest = true
		}
	}
	if input.Capabilities != nil {
		if err := validateCapabilities(current.Type, *input.Capabilities); err != nil {
			return plan, err
		}
		var persisted Capabilities
		if err := json.Unmarshal(current.Capabilities, &persisted); err != nil {
			return plan, err
		}
		if persisted != *input.Capabilities {
			value, err := json.Marshal(*input.Capabilities)
			if err != nil {
				return plan, err
			}
			plan.updates["capabilities"] = datatypes.JSON(value)
			plan.invalidateTest = true
		}
	}
	if input.Enabled != nil && *input.Enabled != current.Enabled {
		plan.updates["enabled"] = *input.Enabled
		plan.disabling = !*input.Enabled
	}
	return plan, nil
}

func (s *Service) DeleteModel(ctx context.Context, modelID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var aiModel model.AIModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&aiModel, "id = ?", modelID).Error; err != nil {
			return notFoundModel(err)
		}

		var bindingCount int64
		if err := tx.Model(&model.PlatformModelBinding{}).
			Where("model_id = ?", modelID).Count(&bindingCount).Error; err != nil {
			return err
		}
		if bindingCount > 0 {
			return apierror.New(
				http.StatusConflict,
				apierror.CodeInvalidInput,
				"已绑定的平台模型不能删除，请先解除绑定",
				nil,
			)
		}

		if err := tx.Where("model_id = ?", modelID).
			Delete(&model.ModelTestRun{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&aiModel)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "AI 模型不存在", nil)
		}
		return nil
	})
}

func (s *Service) TestModel(ctx context.Context, modelID string, adminID string) (*model.AIModel, error) {
	var value model.AIModel
	if err := s.db.WithContext(ctx).First(&value, "id = ?", modelID).Error; err != nil {
		return nil, notFoundModel(err)
	}
	var providerModel model.AIProvider
	if err := s.db.WithContext(ctx).First(&providerModel, "id = ?", value.ProviderID).Error; err != nil {
		return nil, err
	}
	adapter, err := s.factory.FromProvider(providerModel)
	var details any
	if err == nil {
		var calls []provider.CallMetadata
		calls, err = testModelCapabilitiesWithDetails(ctx, adapter, value)
		if len(calls) == 1 {
			details = calls[0]
		} else if len(calls) > 1 {
			details = map[string]any{"calls": calls}
		}
		if err != nil {
			details = providerErrorDetails(err)
		}
	} else {
		details = providerErrorDetails(err)
	}
	status := StatusHealthy
	summary := "模型能力测试正常"
	if err != nil {
		status = StatusError
		summary = "模型测试失败：" + safeError(err)
	}
	now := time.Now().UTC()
	if updateErr := s.db.WithContext(ctx).Model(&value).Updates(map[string]any{
		"test_status":       status,
		"last_tested_at":    &now,
		"last_test_summary": summary,
		"last_test_details": marshalTestDetails(details),
	}).Error; updateErr != nil {
		return nil, updateErr
	}
	_ = s.recordTest(ctx, providerModel.ID, &value.ID, "model", status, summary, value.ConfigVersion, adminID, details)
	value.TestStatus = status
	value.LastTestedAt = &now
	value.LastTestSummary = summary
	value.LastTestDetails = marshalTestDetails(details)
	return &value, nil
}

func (s *Service) Bind(
	ctx context.Context,
	adminID string,
	input BindInput,
	idempotencyKey string,
) (map[string]*string, error) {
	if input.Type != "chat" && input.Type != "image" {
		return nil, apierror.Invalid("平台模型类型无效", nil)
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "manage", PrincipalID: adminID, Method: http.MethodPost,
			Path: "/api/manage/platform-model-bindings", Key: idempotencyKey, Request: input,
		})
		if err != nil {
			return err
		}
		if replay != nil {
			return nil
		}
		if input.ModelID == nil {
			if err := tx.Where("binding_type = ?", input.Type).Delete(&model.PlatformModelBinding{}).Error; err != nil {
				return err
			}
			return idempotency.CompleteTx(
				tx, recordID, http.StatusOK, 0, "platform_model_binding", nil,
				map[string]any{"type": input.Type, "modelId": nil},
			)
		}
		var aiModel model.AIModel
		if err := tx.First(&aiModel, "id = ?", *input.ModelID).Error; err != nil {
			return notFoundModel(err)
		}
		if aiModel.Type != input.Type || !aiModel.Enabled || aiModel.TestStatus != StatusHealthy {
			return apierror.New(http.StatusConflict, apierror.CodeAICapability, "模型类型、状态或测试结果不满足绑定要求", nil)
		}
		var providerModel model.AIProvider
		if err := tx.First(&providerModel, "id = ?", aiModel.ProviderID).Error; err != nil {
			return err
		}
		if !providerModel.Enabled || providerModel.ConnectionStatus != StatusHealthy {
			return apierror.New(http.StatusConflict, apierror.CodeAIUnavailable, "服务商尚未通过连接测试", nil)
		}
		var capabilities Capabilities
		if err := json.Unmarshal(aiModel.Capabilities, &capabilities); err != nil {
			return err
		}
		if err := validatePlatformCapabilities(input.Type, capabilities); err != nil {
			return err
		}
		binding := model.PlatformModelBinding{
			BindingType:           input.Type,
			ProviderID:            providerModel.ID,
			ModelID:               aiModel.ID,
			ProviderConfigVersion: providerModel.ConfigVersion,
			ModelConfigVersion:    aiModel.ConfigVersion,
			BoundBy:               adminID,
			Version:               1,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "binding_type"}},
			DoUpdates: clause.Assignments(map[string]any{
				"provider_id":             binding.ProviderID,
				"model_id":                binding.ModelID,
				"provider_config_version": binding.ProviderConfigVersion,
				"model_config_version":    binding.ModelConfigVersion,
				"bound_by":                binding.BoundBy,
				"version":                 gorm.Expr("platform_model_bindings.version + 1"),
				"updated_at":              time.Now().UTC(),
			}),
		}).Create(&binding).Error; err != nil {
			return err
		}
		return idempotency.CompleteTx(
			tx, recordID, http.StatusOK, 0, "platform_model_binding", &binding.ModelID,
			map[string]any{"type": input.Type, "modelId": binding.ModelID},
		)
	})
	if err != nil {
		return nil, err
	}
	return s.Bindings(ctx)
}

func (s *Service) Bindings(ctx context.Context) (map[string]*string, error) {
	var bindings []model.PlatformModelBinding
	if err := s.db.WithContext(ctx).Find(&bindings).Error; err != nil {
		return nil, err
	}
	result := map[string]*string{"chat": nil, "image": nil}
	for _, binding := range bindings {
		value := binding.ModelID
		result[binding.BindingType] = &value
	}
	return result, nil
}

func validateCapabilities(modelType string, capabilities Capabilities) error {
	switch modelType {
	case "chat":
		if capabilities.TextToImage || capabilities.ImageToImage {
			return apierror.Invalid("聊天模型不能声明生图能力", nil)
		}
	case "image":
		if capabilities.PromptOptimization || capabilities.VisionInput {
			return apierror.Invalid("生图模型不能声明对话能力", nil)
		}
	default:
		return apierror.Invalid("模型类型无效", nil)
	}
	return nil
}

func validatePlatformCapabilities(modelType string, capabilities Capabilities) error {
	if modelType == "chat" && !capabilities.PromptOptimization {
		return apierror.New(http.StatusConflict, apierror.CodeAICapability, "平台对话模型必须支持提示词优化", nil)
	}
	if modelType == "image" && (!capabilities.TextToImage || !capabilities.ImageToImage) {
		return apierror.New(http.StatusConflict, apierror.CodeAICapability, "平台生图模型必须同时支持文生图和图生图", nil)
	}
	return nil
}

func testModelCapabilities(ctx context.Context, adapter provider.Adapter, value model.AIModel) error {
	_, err := testModelCapabilitiesWithDetails(ctx, adapter, value)
	return err
}

func testModelCapabilitiesWithDetails(ctx context.Context, adapter provider.Adapter, value model.AIModel) ([]provider.CallMetadata, error) {
	details := make([]provider.CallMetadata, 0, 2)
	switch value.Type {
	case "chat":
		var capabilities Capabilities
		if err := json.Unmarshal(value.Capabilities, &capabilities); err != nil {
			return details, err
		}
		request := provider.OptimizePromptRequest{
			Model:  value.ModelID,
			Prompt: "Reply with OK.",
		}
		if capabilities.VisionInput {
			sample, err := samplePNG()
			if err != nil {
				return details, err
			}
			request.Prompt = "Look at the image and reply with OK."
			request.ImageDataURLs = []string{
				"data:image/png;base64," + base64.StdEncoding.EncodeToString(sample),
			}
		}
		result, err := adapter.OptimizePrompt(ctx, request)
		if err != nil {
			return details, err
		}
		details = append(details, result.RequestSummary)
		return details, nil
	case "image":
		var capabilities Capabilities
		if err := json.Unmarshal(value.Capabilities, &capabilities); err != nil {
			return details, err
		}
		if capabilities.TextToImage {
			result, err := adapter.TestModel(ctx, provider.ModelTestRequest{
				Model:  value.ModelID,
				Type:   provider.ModelTypeImage,
				Prompt: "A plain white studio card.",
			})
			if err != nil {
				return details, err
			}
			details = append(details, result.RequestSummary)
		}
		if capabilities.ImageToImage {
			sample, err := samplePNG()
			if err != nil {
				return details, err
			}
			imageInput := provider.ImageInput{
				Filename: "capability-test.png", ContentType: "image/png", Reader: bytes.NewReader(sample),
			}
			result, err := adapter.TestModel(ctx, provider.ModelTestRequest{
				Model:  value.ModelID,
				Type:   provider.ModelTypeImage,
				Prompt: "Keep this image unchanged.",
				Image:  &imageInput,
			})
			if err != nil {
				return details, err
			}
			details = append(details, result.RequestSummary)
		}
		return details, nil
	default:
		return details, errors.New("unsupported model type")
	}
}

func samplePNG() ([]byte, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			canvas.Set(x, y, color.White)
		}
	}
	var buffer bytes.Buffer
	err := png.Encode(&buffer, canvas)
	return buffer.Bytes(), err
}

func (s *Service) recordTest(
	ctx context.Context,
	providerID string,
	modelID *string,
	kind string,
	status string,
	summary string,
	version int64,
	adminID string,
	details any,
) error {
	now := time.Now().UTC()
	run := model.ModelTestRun{
		ProviderID:    providerID,
		ModelID:       modelID,
		Kind:          kind,
		Status:        status,
		Summary:       summary,
		Details:       marshalTestDetails(details),
		ConfigVersion: version,
		RequestedBy:   adminID,
		CompletedAt:   &now,
	}
	return s.db.WithContext(ctx).Create(&run).Error
}

func marshalTestDetails(value any) []byte {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return encoded
}

func providerErrorDetails(err error) any {
	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		return map[string]any{"errorKind": "internal"}
	}
	metadata := providerErr.Metadata
	metadata.ErrorKind = string(providerErr.Kind)
	if metadata.Status == 0 {
		metadata.Status = providerErr.StatusCode
	}
	return metadata
}

func validateBaseURL(rawURL string, allowHTTP bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return apierror.Invalid("Base URL 无效", nil)
	}
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return apierror.Invalid("Base URL 必须使用 HTTPS", nil)
	}
	return nil
}

func safeError(err error) string {
	value := strings.TrimSpace(err.Error())
	if len([]rune(value)) > 180 {
		return string([]rune(value)[:180])
	}
	return value
}

func notFoundProvider(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "AI 服务商不存在", nil)
	}
	return err
}

func notFoundModel(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "AI 模型不存在", nil)
	}
	return err
}

var providerCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,39}$`)
