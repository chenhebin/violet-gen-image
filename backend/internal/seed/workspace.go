package seed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/generation"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/prompt"
	"yingyan.local/backend/internal/retouch"
	"yingyan.local/backend/internal/storage"
)

type demoAssetSpec struct {
	name   string
	kind   string
	role   string
	start  color.RGBA
	end    color.RGBA
	accent color.RGBA
}

func seedDemoWorkspace(
	ctx context.Context,
	db *gorm.DB,
	store storage.Store,
	user model.User,
	admin model.AdminAccount,
	provider model.AIProvider,
) error {
	var chatModel model.AIModel
	if err := db.Where("provider_id = ? AND type = ?", provider.ID, "chat").First(&chatModel).Error; err != nil {
		return err
	}
	var imageModel model.AIModel
	if err := db.Where("provider_id = ? AND type = ?", provider.ID, "image").First(&imageModel).Error; err != nil {
		return err
	}

	specs := []demoAssetSpec{
		{
			name: "source-portrait", kind: asset.KindSource,
			start:  color.RGBA{R: 43, G: 52, B: 65, A: 255},
			end:    color.RGBA{R: 197, G: 158, B: 139, A: 255},
			accent: color.RGBA{R: 238, G: 218, B: 201, A: 255},
		},
		{
			name: "style-coast", kind: asset.KindReference, role: "style",
			start:  color.RGBA{R: 43, G: 94, B: 113, A: 255},
			end:    color.RGBA{R: 226, G: 186, B: 128, A: 255},
			accent: color.RGBA{R: 240, G: 235, B: 218, A: 255},
		},
		{
			name: "result-blue", kind: asset.KindAIResult,
			start:  color.RGBA{R: 25, G: 61, B: 87, A: 255},
			end:    color.RGBA{R: 93, G: 142, B: 156, A: 255},
			accent: color.RGBA{R: 226, G: 191, B: 159, A: 255},
		},
		{
			name: "result-coral", kind: asset.KindAIResult,
			start:  color.RGBA{R: 97, G: 54, B: 57, A: 255},
			end:    color.RGBA{R: 204, G: 111, B: 91, A: 255},
			accent: color.RGBA{R: 240, G: 212, B: 184, A: 255},
		},
		{
			name: "result-film", kind: asset.KindAIResult,
			start:  color.RGBA{R: 38, G: 48, B: 42, A: 255},
			end:    color.RGBA{R: 135, G: 143, B: 111, A: 255},
			accent: color.RGBA{R: 231, G: 210, B: 164, A: 255},
		},
	}
	assets := make(map[string]model.Asset, len(specs))
	for _, spec := range specs {
		value, err := upsertDemoAsset(ctx, db, store, user.ID, spec)
		if err != nil {
			return err
		}
		assets[spec.name] = value
	}

	sections := prompt.Sections{
		Subject:     "保留人物真实五官和自然肤质",
		Scene:       "傍晚海岸，背景简洁，空气通透",
		Style:       "克制的电影胶片质感，青绿色阴影与暖色肤色",
		Composition: "竖幅半身肖像，视线自然，保留适度环境空间",
		Details:     "发丝清晰，衣物纹理自然，光影柔和不过度磨皮",
		Negative:    "避免塑料皮肤、五官变形、额外肢体、文字和水印",
		Output:      "高质量写实人像，适合社交平台发布",
	}
	firstPrompt, err := ensureDemoPrompt(
		db, user.ID, provider, chatModel,
		"把原图修成傍晚海边的电影感人像，保留本人五官和自然肤质。",
		assets["source-portrait"], assets["style-coast"], sections,
	)
	if err != nil {
		return err
	}
	firstTask, err := ensureDemoTask(
		db, user.ID, provider, imageModel, firstPrompt,
		"海边电影感人像",
		[]model.Asset{assets["source-portrait"], assets["style-coast"]},
		[]model.Asset{assets["result-blue"], assets["result-coral"]},
	)
	if err != nil {
		return err
	}

	secondSections := sections
	secondSections.Style = "低饱和蓝调棚拍质感，层次清晰，光影克制"
	secondPrompt, err := ensureDemoPrompt(
		db, user.ID, provider, chatModel,
		"调整成高级蓝调肖像，人物更突出，背景干净但不要失去真实感。",
		assets["source-portrait"], assets["style-coast"], secondSections,
	)
	if err != nil {
		return err
	}
	secondTask, err := ensureDemoTask(
		db, user.ID, provider, imageModel, secondPrompt,
		"蓝调肖像精修",
		[]model.Asset{assets["source-portrait"], assets["style-coast"]},
		[]model.Asset{assets["result-film"]},
	)
	if err != nil {
		return err
	}
	if err := ensureDemoRetouchTicket(
		db, user, admin, secondTask, assets["result-film"],
	); err != nil {
		return err
	}

	_ = firstTask
	return nil
}

func upsertDemoAsset(
	ctx context.Context,
	db *gorm.DB,
	store storage.Store,
	userID string,
	spec demoAssetSpec,
) (model.Asset, error) {
	const width = 600
	const height = 800
	data, err := makeDemoPNG(width, height, spec.start, spec.end, spec.accent)
	if err != nil {
		return model.Asset{}, err
	}
	objectKey := "demo/" + spec.name + ".png"
	if _, err := store.Put(
		ctx,
		objectKey,
		bytes.NewReader(data),
		int64(len(data)),
		storage.PutOptions{ContentType: "image/png", CacheControl: "private, max-age=0"},
	); err != nil {
		return model.Asset{}, err
	}
	sum := sha256.Sum256(data)
	var value model.Asset
	err = db.Where("object_key = ?", objectKey).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		value = model.Asset{
			OwnerUserID:   &userID,
			Kind:          spec.kind,
			ReferenceRole: spec.role,
			OriginalName:  spec.name + ".png",
			MIMEType:      "image/png",
			SizeBytes:     int64(len(data)),
			Width:         width,
			Height:        height,
			SHA256:        hex.EncodeToString(sum[:]),
			Bucket:        store.Bucket(),
			ObjectKey:     objectKey,
			Version:       1,
		}
		return value, db.Create(&value).Error
	}
	if err != nil {
		return model.Asset{}, err
	}
	err = db.Model(&value).Updates(map[string]any{
		"owner_user_id": userID, "kind": spec.kind, "reference_role": spec.role,
		"original_name": spec.name + ".png", "mime_type": "image/png",
		"size_bytes": int64(len(data)), "width": width, "height": height,
		"sha256": hex.EncodeToString(sum[:]), "bucket": store.Bucket(),
		"cleaned_at": nil, "cleanup_reason": "",
	}).Error
	return value, err
}

func ensureDemoPrompt(
	db *gorm.DB,
	userID string,
	provider model.AIProvider,
	chatModel model.AIModel,
	source string,
	sourceAsset model.Asset,
	referenceAsset model.Asset,
	sections prompt.Sections,
) (model.PromptVersion, error) {
	var value model.PromptVersion
	err := db.Where(
		"user_id = ? AND source = ? AND status = ?",
		userID, source, prompt.StatusConfirmed,
	).First(&value).Error
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.PromptVersion{}, err
	}
	sectionJSON, _ := json.Marshal(sections)
	sourceJSON, _ := json.Marshal([]string{sourceAsset.ID})
	referenceJSON, _ := json.Marshal([]prompt.ReferenceAsset{{
		AssetID: referenceAsset.ID,
		Role:    referenceAsset.ReferenceRole,
	}})
	now := time.Now().UTC()
	value = model.PromptVersion{
		UserID: userID, Source: source, Mode: "image-to-image",
		Sections:        datatypes.JSON(sectionJSON),
		SourceAssetIDs:  datatypes.JSON(sourceJSON),
		ReferenceAssets: datatypes.JSON(referenceJSON),
		ProviderID:      &provider.ID, ModelID: &chatModel.ID,
		ProviderConfigVersion: provider.ConfigVersion,
		ModelConfigVersion:    chatModel.ConfigVersion,
		Status:                prompt.StatusConfirmed, ConfirmedAt: &now,
	}
	return value, db.Create(&value).Error
}

func ensureDemoTask(
	db *gorm.DB,
	userID string,
	provider model.AIProvider,
	imageModel model.AIModel,
	promptVersion model.PromptVersion,
	title string,
	inputAssets []model.Asset,
	outputAssets []model.Asset,
) (model.GenerationTask, error) {
	var existing model.GenerationTask
	err := db.Where("user_id = ? AND title = ?", userID, title).First(&existing).Error
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.GenerationTask{}, err
	}

	taskID := uuid.NewString()
	creditService := credit.New(db)
	reservation, _, err := creditService.ReserveTx(
		db, userID, "generation", taskID, len(outputAssets), "演示 AI 生成任务预占",
	)
	if err != nil {
		return model.GenerationTask{}, err
	}
	if err := creditService.SettleTx(db, reservation.ID, len(outputAssets)); err != nil {
		return model.GenerationTask{}, err
	}
	settings, _ := json.Marshal(generation.Settings{
		AspectRatio: "3:4", OutputCount: len(outputAssets), ReferenceStrength: 68,
	})
	startedAt := time.Now().UTC().Add(-20 * time.Minute)
	completedAt := startedAt.Add(2 * time.Minute)
	task := model.GenerationTask{
		BaseModel: model.BaseModel{ID: taskID},
		UserID:    userID, PromptVersionID: promptVersion.ID, Title: title,
		Mode: promptVersion.Mode, Status: generation.StatusCompleted,
		Settings: datatypes.JSON(settings), OutputCount: len(outputAssets),
		CompletedOutputs: len(outputAssets), ReservedCredits: len(outputAssets),
		SpentCredits: len(outputAssets), CreditReservationID: reservation.ID,
		ProviderID: provider.ID, ModelID: imageModel.ID,
		ProviderNameSnapshot:     provider.Name,
		ProviderBaseURLSnapshot:  provider.BaseURL,
		APIKeyCiphertextSnapshot: append([]byte(nil), provider.APIKeyCiphertext...),
		ModelNameSnapshot:        imageModel.ModelID,
		ModelDisplayNameSnapshot: imageModel.DisplayName,
		ProviderConfigVersion:    provider.ConfigVersion,
		ModelConfigVersion:       imageModel.ConfigVersion,
		StartedAt:                &startedAt, CompletedAt: &completedAt, Version: 1,
	}
	if err := db.Create(&task).Error; err != nil {
		return model.GenerationTask{}, err
	}
	for _, input := range inputAssets {
		if err := db.Create(&model.GenerationTaskAsset{
			TaskID: task.ID, AssetID: input.ID,
			Usage: input.Kind, ReferenceRole: input.ReferenceRole,
		}).Error; err != nil {
			return model.GenerationTask{}, err
		}
		if err := db.Create(&model.AssetRelation{
			AssetID: input.ID, ResourceType: "generation_task",
			ResourceID: task.ID, RelationType: input.Kind,
		}).Error; err != nil {
			return model.GenerationTask{}, err
		}
	}
	for index, outputAsset := range outputAssets {
		output := model.GenerationOutput{
			TaskID: task.ID, OutputIndex: index, Status: generation.OutputSucceeded,
			AssetID: &outputAsset.ID, ProviderResponseID: fmt.Sprintf("demo-output-%d", index+1),
			StartedAt: &startedAt, CompletedAt: &completedAt, Version: 1,
		}
		if err := db.Create(&output).Error; err != nil {
			return model.GenerationTask{}, err
		}
		job := model.GenerationJob{
			JobType: "generation_output", TaskID: task.ID, OutputID: &output.ID,
			Status: generation.JobCompleted, Attempts: 1, MaxAttempts: 1,
			AvailableAt: startedAt, LockedBy: "demo-seed",
			LockedAt: &startedAt, HeartbeatAt: &completedAt,
			StartedAt: &startedAt, CompletedAt: &completedAt, Version: 1,
		}
		if err := db.Create(&job).Error; err != nil {
			return model.GenerationTask{}, err
		}
		accepted := true
		if err := db.Create(&model.ProviderAttempt{
			JobID: job.ID, ProviderID: provider.ID, ModelID: imageModel.ID,
			AttemptNo: 1, Status: "succeeded",
			Operation: "edit_image", HTTPMethod: "POST", EndpointPath: "/v1/images/edits",
			ModelName: imageModel.ModelID, ResponseStatus: 200,
			ExternalRequestID: output.ProviderResponseID,
			RequestAccepted:   &accepted, LatencyMillis: 1300,
			RequestSummary: datatypes.JSON([]byte(`{"operation":"edit_image","method":"POST","path":"/v1/images/edits","model":"gpt-image-2","parameterSummary":{"promptLength":96,"imageCount":1,"outputCount":1},"status":200,"latencyMs":1300}`)),
			ResponseMetadata: datatypes.JSON([]byte(`{"imageCount":1,"contentType":"image/png","source":"base64"}`)),
			StartedAt: startedAt, CompletedAt: &completedAt,
		}).Error; err != nil {
			return model.GenerationTask{}, err
		}
		if err := db.Create(&model.AssetRelation{
			AssetID: outputAsset.ID, ResourceType: "generation_task",
			ResourceID: task.ID, RelationType: "result",
		}).Error; err != nil {
			return model.GenerationTask{}, err
		}
	}
	return task, nil
}

func ensureDemoRetouchTicket(
	db *gorm.DB,
	user model.User,
	admin model.AdminAccount,
	task model.GenerationTask,
	selectedResult model.Asset,
) error {
	var count int64
	if err := db.Model(&model.RetouchTicket{}).
		Where("task_id = ?", task.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	ticket := model.RetouchTicket{
		TicketNo: "RT-DEMO-0001", UserID: user.ID, TaskID: task.ID,
		Status:       retouch.StatusSubmitted,
		Requirements: "AI 成片整体满意，希望人工微调发丝边缘和面部光影，保持本人特征。",
		Version:      1,
	}
	if err := db.Create(&ticket).Error; err != nil {
		return err
	}
	if err := db.Create(&model.RetouchSelectedResult{
		TicketID: ticket.ID, AssetID: selectedResult.ID,
	}).Error; err != nil {
		return err
	}
	if err := db.Create(&model.AssetRelation{
		AssetID: selectedResult.ID, ResourceType: "retouch_ticket",
		ResourceID: ticket.ID, RelationType: "selected_result",
	}).Error; err != nil {
		return err
	}
	if err := db.Create(&model.RetouchEvent{
		TicketID: ticket.ID, ToStatus: retouch.StatusSubmitted,
		Action: "submit", ActorRealm: "user", ActorID: user.ID,
		Summary: "用户提交人工修图需求",
	}).Error; err != nil {
		return err
	}
	now := time.Now().UTC().Add(-5 * time.Minute)
	quote := model.RetouchQuote{
		TicketID: ticket.ID, QuoteVersion: 1, Credits: 3,
		Notes:  "精修发丝、肤色和局部光影，预计 24 小时内交付。",
		Status: "active", CreatedBy: admin.ID,
	}
	if err := db.Create(&quote).Error; err != nil {
		return err
	}
	if err := db.Model(&ticket).Updates(map[string]any{
		"status":           retouch.StatusQuotePending,
		"current_quote_id": quote.ID,
		"quoted_at":        &now,
		"version":          gorm.Expr("version + 1"),
	}).Error; err != nil {
		return err
	}
	if err := db.Create(&model.RetouchEvent{
		TicketID: ticket.ID, FromStatus: retouch.StatusSubmitted,
		ToStatus: retouch.StatusQuotePending, Action: "quote",
		ActorRealm: "manage", ActorID: admin.ID, Summary: quote.Notes,
	}).Error; err != nil {
		return err
	}
	after, _ := json.Marshal(map[string]any{"credits": quote.Credits})
	resourceID := ticket.ID
	adminID := admin.ID
	return db.Create(&model.AuditLog{
		AdminID: &adminID, AdminEmail: admin.Email, AdminRole: admin.Role,
		Action: "retouch.quote", ResourceType: "retouch_ticket", ResourceID: &resourceID,
		AfterSummary: datatypes.JSON(after), Result: "success",
		RequestID: "demo-seed-retouch-quote", CreatedAt: now,
	}).Error
}

func makeDemoPNG(
	width int,
	height int,
	start color.RGBA,
	end color.RGBA,
	accent color.RGBA,
) ([]byte, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			base := color.RGBA{
				R: blend(start.R, end.R, y, height-1),
				G: blend(start.G, end.G, y, height-1),
				B: blend(start.B, end.B, y, height-1),
				A: 255,
			}
			dx := x - width/2
			dy := y - height*2/5
			if dx*dx*4+dy*dy < width*width/5 {
				base = color.RGBA{
					R: blend(base.R, accent.R, 2, 5),
					G: blend(base.G, accent.G, 2, 5),
					B: blend(base.B, accent.B, 2, 5),
					A: 255,
				}
			}
			canvas.SetRGBA(x, y, base)
		}
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func blend(start uint8, end uint8, position int, length int) uint8 {
	if length <= 0 {
		return start
	}
	return uint8((int(start)*(length-position) + int(end)*position) / length)
}
