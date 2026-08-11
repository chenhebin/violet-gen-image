package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/idempotency"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/prompt"
)

const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusPartial    = "partial"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"

	OutputQueued     = "queued"
	OutputProcessing = "processing"
	OutputSucceeded  = "succeeded"
	OutputFailed     = "failed"
	OutputCancelled  = "cancelled"

	JobQueued     = "queued"
	JobProcessing = "processing"
	JobCompleted  = "completed"
	JobFailed     = "failed"
	JobCancelled  = "cancelled"
)

type Service struct {
	db      *gorm.DB
	credits *credit.Service
	assets  *asset.Service
	prompts *prompt.Service
}

func New(db *gorm.DB, credits *credit.Service, assets *asset.Service, prompts *prompt.Service) *Service {
	return &Service{db: db, credits: credits, assets: assets, prompts: prompts}
}

func (s *Service) Create(
	ctx context.Context,
	userID string,
	input CreateInput,
	idempotencyKey string,
) (*TaskDTO, error) {
	if err := validateSettings(input.Settings); err != nil {
		return nil, err
	}
	input.AssetIDs = uniqueStrings(input.AssetIDs)
	if len(input.AssetIDs) > 8 {
		return nil, apierror.Invalid("生成素材不能超过 8 张", nil)
	}

	var taskID string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "user",
			PrincipalID:    userID,
			Method:         http.MethodPost,
			Path:           "/api/generations",
			Key:            idempotencyKey,
			Request:        input,
		})
		if err != nil {
			return err
		}
		if replay != nil {
			var reference struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(replay.Data, &reference); err != nil {
				return err
			}
			taskID = reference.ID
			return nil
		}

		assets, err := loadOwnedAssets(tx, userID, input.AssetIDs)
		if err != nil {
			return err
		}
		resolvedMode := generationModeForAssets(assets)
		promptVersion, referenceRoles, err := resolvePromptVersion(
			tx,
			userID,
			input,
			assets,
			resolvedMode,
		)
		if err != nil {
			return err
		}

		providerModel, aiModel, err := currentImageModel(tx, resolvedMode)
		if err != nil {
			return err
		}
		settingsJSON, _ := json.Marshal(input.Settings)
		taskID = uuid.NewString()
		reservation, _, err := s.credits.ReserveTx(
			tx,
			userID,
			"generation",
			taskID,
			input.Settings.OutputCount,
			"AI 生成任务预占",
		)
		if err != nil {
			return err
		}

		task := model.GenerationTask{
			BaseModel:                model.BaseModel{ID: taskID},
			UserID:                   userID,
			PromptVersionID:          promptVersion.ID,
			Title:                    taskTitle(promptVersion.Source),
			Mode:                     resolvedMode,
			Status:                   StatusQueued,
			Settings:                 datatypes.JSON(settingsJSON),
			OutputCount:              input.Settings.OutputCount,
			ReservedCredits:          input.Settings.OutputCount,
			CreditReservationID:      reservation.ID,
			ProviderID:               providerModel.ID,
			ModelID:                  aiModel.ID,
			ProviderNameSnapshot:     providerModel.Name,
			ProviderBaseURLSnapshot:  providerModel.BaseURL,
			APIKeyCiphertextSnapshot: append([]byte(nil), providerModel.APIKeyCiphertext...),
			ModelNameSnapshot:        aiModel.ModelID,
			ModelDisplayNameSnapshot: aiModel.DisplayName,
			ProviderConfigVersion:    providerModel.ConfigVersion,
			ModelConfigVersion:       aiModel.ConfigVersion,
			Version:                  1,
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		for _, assetModel := range assets {
			usage := assetModel.Kind
			referenceRole := assetModel.ReferenceRole
			if assetModel.Kind == asset.KindReference {
				referenceRole = referenceRoles[assetModel.ID]
				if referenceRole == "" {
					return apierror.Invalid(
						"参考图用途快照无效，请重新优化并确认提示词",
						map[string]string{"assetId": assetModel.ID},
					)
				}
			}
			relation := model.GenerationTaskAsset{
				TaskID:        task.ID,
				AssetID:       assetModel.ID,
				Usage:         usage,
				ReferenceRole: referenceRole,
			}
			if err := tx.Create(&relation).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.AssetRelation{
				AssetID:      assetModel.ID,
				ResourceType: "generation_task",
				ResourceID:   task.ID,
				RelationType: usage,
			}).Error; err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		for index := 0; index < input.Settings.OutputCount; index++ {
			output := model.GenerationOutput{
				TaskID: task.ID, OutputIndex: index, Status: OutputQueued, Version: 1,
			}
			if err := tx.Create(&output).Error; err != nil {
				return err
			}
			job := model.GenerationJob{
				JobType:     "generation_output",
				TaskID:      task.ID,
				OutputID:    &output.ID,
				Status:      JobQueued,
				MaxAttempts: 1,
				AvailableAt: now,
				Version:     1,
			}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
		}
		reference := struct {
			ID string `json:"id"`
		}{ID: task.ID}
		return idempotency.CompleteTx(tx, recordID, http.StatusAccepted, 0, "generation_task", &task.ID, reference)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, taskID)
}

func (s *Service) Get(ctx context.Context, userID string, taskID string) (*TaskDTO, error) {
	var task model.GenerationTask
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", taskID, userID).
		First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierror.New(http.StatusNotFound, apierror.CodeTaskNotFound, "任务不存在", nil)
	}
	if err != nil {
		return nil, err
	}
	return s.dto(ctx, task)
}

func (s *Service) List(ctx context.Context, userID string) ([]TaskDTO, error) {
	var tasks []model.GenerationTask
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	result := make([]TaskDTO, 0, len(tasks))
	for _, task := range tasks {
		value, err := s.dto(ctx, task)
		if err != nil {
			return nil, err
		}
		result = append(result, *value)
	}
	return result, nil
}

func (s *Service) Cancel(ctx context.Context, userID string, taskID string) (*TaskDTO, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.GenerationTask
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", taskID, userID).
			First(&task).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierror.New(http.StatusNotFound, apierror.CodeTaskNotFound, "任务不存在", nil)
		}
		if err != nil {
			return err
		}
		if task.Status == StatusCancelled {
			return nil
		}
		if task.Status != StatusQueued {
			return apierror.New(http.StatusConflict, apierror.CodeInvalidInput, "只有排队中的任务可以取消", nil)
		}
		if _, err := s.credits.ReleaseTx(
			tx,
			task.CreditReservationID,
			task.ReservedCredits,
			"用户取消排队任务",
		); err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&task).Updates(map[string]any{
			"status":           StatusCancelled,
			"refunded_credits": task.ReservedCredits,
			"cancelled_at":     &now,
			"completed_at":     &now,
			"version":          gorm.Expr("version + 1"),
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.GenerationOutput{}).Where("task_id = ? AND status = ?", task.ID, OutputQueued).
			Update("status", OutputCancelled).Error; err != nil {
			return err
		}
		return tx.Model(&model.GenerationJob{}).Where("task_id = ? AND status = ?", task.ID, JobQueued).
			Updates(map[string]any{"status": JobCancelled, "completed_at": &now}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, taskID)
}

func (s *Service) dto(ctx context.Context, task model.GenerationTask) (*TaskDTO, error) {
	var promptVersion model.PromptVersion
	if err := s.db.WithContext(ctx).First(&promptVersion, "id = ?", task.PromptVersionID).Error; err != nil {
		return nil, err
	}
	var sections prompt.Sections
	if err := json.Unmarshal(promptVersion.Sections, &sections); err != nil {
		return nil, err
	}
	var settings Settings
	if err := json.Unmarshal(task.Settings, &settings); err != nil {
		return nil, err
	}
	var taskAssets []model.GenerationTaskAsset
	if err := s.db.WithContext(ctx).Where("task_id = ?", task.ID).Find(&taskAssets).Error; err != nil {
		return nil, err
	}
	assets := make([]asset.DTO, 0, len(taskAssets))
	for _, link := range taskAssets {
		assetModel, err := s.assets.GetByID(ctx, link.AssetID)
		if err != nil {
			return nil, err
		}
		value, err := s.assets.DTO(ctx, *assetModel)
		if err != nil {
			return nil, err
		}
		if link.Usage == asset.KindReference {
			value.Role = link.ReferenceRole
		}
		assets = append(assets, value)
	}
	var outputs []model.GenerationOutput
	if err := s.db.WithContext(ctx).
		Where("task_id = ? AND status = ?", task.ID, OutputSucceeded).
		Order("output_index ASC").
		Find(&outputs).Error; err != nil {
		return nil, err
	}
	results := make([]ResultDTO, 0, len(outputs))
	for _, output := range outputs {
		if output.AssetID == nil {
			continue
		}
		assetModel, err := s.assets.GetByID(ctx, *output.AssetID)
		if err != nil {
			return nil, err
		}
		value, err := s.assets.DTO(ctx, *assetModel)
		if err != nil {
			return nil, err
		}
		downloadURL, err := s.assets.DownloadURL(ctx, *assetModel)
		if err != nil {
			return nil, err
		}
		results = append(results, ResultDTO{
			ID: assetModel.ID, URL: value.PreviewURL, DownloadURL: downloadURL,
			Width: assetModel.Width, Height: assetModel.Height,
		})
	}
	progress := EstimatedProgress(task, time.Now().UTC())
	value := &TaskDTO{
		ID:     task.ID,
		Mode:   task.Mode,
		Title:  task.Title,
		Status: task.Status,
		Prompt: prompt.DTO{
			ID: task.PromptVersionID, Source: promptVersion.Source, Sections: sections, ConfirmedAt: promptVersion.ConfirmedAt,
		},
		Settings:        settings,
		Assets:          assets,
		RequestedCount:  task.OutputCount,
		SuccessfulCount: task.CompletedOutputs,
		ReservedCredits: task.ReservedCredits,
		SpentCredits:    task.SpentCredits,
		RefundedCredits: task.RefundedCredits,
		Progress:        progress,
		Results:         results,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
	var ticket model.RetouchTicket
	if err := s.db.WithContext(ctx).Where("task_id = ?", task.ID).
		Order("created_at DESC").First(&ticket).Error; err == nil {
		var quoteCredits *int
		if ticket.CurrentQuoteID != nil {
			var quote model.RetouchQuote
			if s.db.WithContext(ctx).First(&quote, "id = ?", *ticket.CurrentQuoteID).Error == nil {
				credits := quote.Credits
				quoteCredits = &credits
			}
		}
		value.RetouchTicket = &RetouchSummary{
			ID: ticket.ID, TicketNo: ticket.TicketNo, Status: ticket.Status,
			UpdatedAt: ticket.UpdatedAt, QuoteCredits: quoteCredits,
		}
	}
	return value, nil
}

func validateSettings(settings Settings) error {
	switch settings.AspectRatio {
	case "1:1", "3:4", "4:3", "9:16", "16:9":
	default:
		return apierror.Invalid("图片比例无效", nil)
	}
	if settings.OutputCount < 1 || settings.OutputCount > 4 {
		return apierror.Invalid("输出数量必须为 1 到 4", nil)
	}
	if settings.ReferenceStrength < 0 || settings.ReferenceStrength > 100 {
		return apierror.Invalid("参考强度必须为 0 到 100", nil)
	}
	return nil
}

func resolvePromptVersion(
	tx *gorm.DB,
	userID string,
	input CreateInput,
	assets []model.Asset,
	resolvedMode string,
) (model.PromptVersion, map[string]string, error) {
	promptVersionID := strings.TrimSpace(input.PromptVersionID)
	source := strings.TrimSpace(input.Source)
	if (promptVersionID == "") == (source == "") {
		return model.PromptVersion{}, nil, apierror.Invalid(
			"请选择已确认的提示词方案，或直接填写画面需求",
			nil,
		)
	}

	if promptVersionID != "" {
		if len(input.ReferenceAssets) > 0 {
			return model.PromptVersion{}, nil, apierror.Invalid("已确认方案不能重复提交参考图用途", nil)
		}
		var version model.PromptVersion
		err := tx.Where("id = ? AND user_id = ? AND status = ?", promptVersionID, userID, prompt.StatusConfirmed).
			First(&version).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.PromptVersion{}, nil, apierror.New(
				http.StatusNotFound,
				apierror.CodePromptNotFound,
				"已确认提示词版本不存在",
				nil,
			)
		}
		if err != nil {
			return model.PromptVersion{}, nil, err
		}
		expectedAssetIDs, err := promptAssetIDs(version)
		if err != nil {
			return model.PromptVersion{}, nil, err
		}
		if !sameStrings(expectedAssetIDs, input.AssetIDs) || version.Mode != resolvedMode {
			return model.PromptVersion{}, nil, apierror.Invalid("素材已经变化，请重新优化并确认提示词", nil)
		}
		referenceRoles, err := promptReferenceRoles(version)
		return version, referenceRoles, err
	}

	return createDirectPromptVersion(tx, userID, source, input.ReferenceAssets, assets, resolvedMode)
}

func createDirectPromptVersion(
	tx *gorm.DB,
	userID string,
	source string,
	referenceInput []prompt.ReferenceAsset,
	assets []model.Asset,
	resolvedMode string,
) (model.PromptVersion, map[string]string, error) {
	version, referenceRoles, err := buildDirectPromptVersion(
		userID,
		source,
		referenceInput,
		assets,
		resolvedMode,
		time.Now().UTC(),
	)
	if err != nil {
		return model.PromptVersion{}, nil, err
	}
	if err := tx.Create(&version).Error; err != nil {
		return model.PromptVersion{}, nil, err
	}
	return version, referenceRoles, nil
}

func buildDirectPromptVersion(
	userID string,
	source string,
	referenceInput []prompt.ReferenceAsset,
	assets []model.Asset,
	resolvedMode string,
	now time.Time,
) (model.PromptVersion, map[string]string, error) {
	length := len([]rune(source))
	if length < 4 || length > 2000 {
		return model.PromptVersion{}, nil, apierror.Invalid("需求描述长度需为 4 到 2000 字", nil)
	}

	sourceIDs := make([]string, 0, len(assets))
	referenceIDs := make(map[string]struct{}, len(assets))
	for _, assetModel := range assets {
		switch assetModel.Kind {
		case asset.KindSource:
			sourceIDs = append(sourceIDs, assetModel.ID)
		case asset.KindReference:
			referenceIDs[assetModel.ID] = struct{}{}
		default:
			return model.PromptVersion{}, nil, apierror.Invalid(
				"生成素材用途无效",
				map[string]string{"assetId": assetModel.ID},
			)
		}
	}

	referenceRoles := make(map[string]string, len(referenceInput))
	references := make([]prompt.ReferenceAsset, 0, len(referenceInput))
	for _, reference := range referenceInput {
		if _, exists := referenceIDs[reference.AssetID]; !exists ||
			!prompt.IsValidReferenceRole(reference.Role) {
			return model.PromptVersion{}, nil, apierror.Invalid(
				"参考图素材或用途无效",
				map[string]string{"assetId": reference.AssetID},
			)
		}
		if _, duplicated := referenceRoles[reference.AssetID]; duplicated {
			return model.PromptVersion{}, nil, apierror.Invalid("参考图用途不能重复", nil)
		}
		referenceRoles[reference.AssetID] = reference.Role
		references = append(references, reference)
	}
	if len(referenceRoles) != len(referenceIDs) {
		return model.PromptVersion{}, nil, apierror.Invalid("请为每张参考图选择用途", nil)
	}

	sectionsJSON, _ := json.Marshal(prompt.Sections{})
	sourceIDsJSON, _ := json.Marshal(sourceIDs)
	referencesJSON, _ := json.Marshal(references)
	version := model.PromptVersion{
		UserID:          userID,
		Source:          source,
		Mode:            resolvedMode,
		Sections:        datatypes.JSON(sectionsJSON),
		SourceAssetIDs:  datatypes.JSON(sourceIDsJSON),
		ReferenceAssets: datatypes.JSON(referencesJSON),
		Status:          prompt.StatusConfirmed,
		ConfirmedAt:     &now,
	}
	return version, referenceRoles, nil
}

func promptAssetIDs(version model.PromptVersion) ([]string, error) {
	var sourceIDs []string
	var references []prompt.ReferenceAsset
	if err := json.Unmarshal(version.SourceAssetIDs, &sourceIDs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(version.ReferenceAssets, &references); err != nil {
		return nil, err
	}
	result := append([]string(nil), sourceIDs...)
	for _, reference := range references {
		result = append(result, reference.AssetID)
	}
	return uniqueStrings(result), nil
}

func promptReferenceRoles(version model.PromptVersion) (map[string]string, error) {
	var references []prompt.ReferenceAsset
	if err := json.Unmarshal(version.ReferenceAssets, &references); err != nil {
		return nil, err
	}
	roles := make(map[string]string, len(references))
	for _, reference := range references {
		roles[reference.AssetID] = reference.Role
	}
	return roles, nil
}

func loadOwnedAssets(tx *gorm.DB, userID string, assetIDs []string) ([]model.Asset, error) {
	if len(assetIDs) == 0 {
		return []model.Asset{}, nil
	}
	var assets []model.Asset
	if err := tx.Where("owner_user_id = ? AND id IN ? AND cleaned_at IS NULL", userID, assetIDs).
		Find(&assets).Error; err != nil {
		return nil, err
	}
	if len(assets) != len(assetIDs) {
		return nil, apierror.New(http.StatusNotFound, apierror.CodeAssetNotFound, "素材不存在", nil)
	}
	return assets, nil
}

func currentImageModel(tx *gorm.DB, mode string) (model.AIProvider, model.AIModel, error) {
	var binding model.PlatformModelBinding
	if err := tx.Where("binding_type = ?", "image").First(&binding).Error; err != nil {
		return model.AIProvider{}, model.AIModel{}, apierror.New(http.StatusServiceUnavailable, apierror.CodeAIUnavailable, "平台生图模型尚未配置", nil)
	}
	var providerModel model.AIProvider
	var aiModel model.AIModel
	if err := tx.First(&providerModel, "id = ?", binding.ProviderID).Error; err != nil {
		return model.AIProvider{}, model.AIModel{}, apierror.New(http.StatusServiceUnavailable, apierror.CodeAIUnavailable, "AI 服务商不可用", nil)
	}
	if err := tx.First(&aiModel, "id = ?", binding.ModelID).Error; err != nil {
		return model.AIProvider{}, model.AIModel{}, apierror.New(http.StatusServiceUnavailable, apierror.CodeAIUnavailable, "AI 模型不可用", nil)
	}
	if !providerModel.Enabled || providerModel.ConnectionStatus != "healthy" ||
		!aiModel.Enabled || aiModel.TestStatus != "healthy" ||
		providerModel.ConfigVersion != binding.ProviderConfigVersion ||
		aiModel.ConfigVersion != binding.ModelConfigVersion {
		return model.AIProvider{}, model.AIModel{}, apierror.New(http.StatusServiceUnavailable, apierror.CodeAIUnavailable, "平台生图模型需要重新测试", nil)
	}
	capabilities := map[string]bool{}
	if err := json.Unmarshal(aiModel.Capabilities, &capabilities); err != nil {
		return model.AIProvider{}, model.AIModel{}, err
	}
	required := "textToImage"
	if mode == "image-to-image" {
		required = "imageToImage"
	}
	if !capabilities[required] {
		return model.AIProvider{}, model.AIModel{}, apierror.New(http.StatusConflict, apierror.CodeAICapability, "当前模型不支持所选创作模式", nil)
	}
	return providerModel, aiModel, nil
}

func generationModeForAssets(assets []model.Asset) string {
	if len(assets) > 0 {
		return "image-to-image"
	}
	return "text-to-image"
}

func taskTitle(source string) string {
	value := []rune(strings.TrimSpace(source))
	if len(value) > 24 {
		return string(value[:24]) + "..."
	}
	if len(value) == 0 {
		return "未命名创作"
	}
	return string(value)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sameStrings(left, right []string) bool {
	left = uniqueStrings(left)
	right = uniqueStrings(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func SizeForAspectRatio(ratio string) string {
	switch ratio {
	case "3:4", "9:16":
		return "1024x1536"
	case "4:3", "16:9":
		return "1536x1024"
	default:
		return "1024x1024"
	}
}

func safeGenerationError(err error) string {
	value := strings.TrimSpace(err.Error())
	if len([]rune(value)) > 300 {
		return string([]rune(value)[:300])
	}
	return value
}

var _ = fmt.Sprintf
