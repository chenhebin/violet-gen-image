package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/ai"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/prompt"
	"yingyan.local/backend/internal/provider"
)

type Processor struct {
	db         *gorm.DB
	credits    *credit.Service
	assets     *asset.Service
	factory    *ai.Factory
	staleAfter time.Duration
	logger     *slog.Logger
}

func NewProcessor(
	db *gorm.DB,
	credits *credit.Service,
	assets *asset.Service,
	factory *ai.Factory,
	staleAfter time.Duration,
	logger *slog.Logger,
) *Processor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Processor{
		db: db, credits: credits, assets: assets, factory: factory,
		staleAfter: staleAfter, logger: logger,
	}
}

func (p *Processor) ProcessNext(ctx context.Context, workerID string) (bool, error) {
	recovered, err := p.recoverStale(ctx, workerID)
	if err != nil || recovered {
		return recovered, err
	}
	job, err := p.claim(ctx, workerID)
	if err != nil || job == nil {
		return false, err
	}
	p.logger.Info(
		"generation_job_claimed",
		"worker_id", workerID,
		"job_id", job.ID,
		"task_id", job.TaskID,
		"output_id", job.OutputID,
	)
	if err := p.execute(ctx, *job); err != nil {
		if finalizeErr := p.fail(ctx, *job, err); finalizeErr != nil {
			return true, fmt.Errorf("execute job: %v; finalize failure: %w", err, finalizeErr)
		}
	}
	return true, nil
}

func (p *Processor) recoverStale(ctx context.Context, workerID string) (bool, error) {
	if p.staleAfter <= 0 {
		return false, nil
	}

	var uncertain *model.GenerationJob
	requeued := false
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job model.GenerationJob
		cutoff := time.Now().UTC().Add(-p.staleAfter)
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"status = ? AND COALESCE(heartbeat_at, locked_at, started_at, created_at) < ?",
				JobProcessing,
				cutoff,
			).
			Order("created_at ASC").
			First(&job).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		var attempts int64
		if err := tx.Model(&model.ProviderAttempt{}).
			Where("job_id = ?", job.ID).
			Count(&attempts).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if attempts == 0 {
			if err := tx.Model(&job).Updates(map[string]any{
				"status":       JobQueued,
				"locked_by":    "",
				"locked_at":    nil,
				"heartbeat_at": nil,
				"started_at":   nil,
				"available_at": now,
				"attempts":     gorm.Expr("GREATEST(attempts - 1, 0)"),
				"version":      gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			if job.OutputID != nil {
				if err := tx.Model(&model.GenerationOutput{}).
					Where("id = ? AND status = ?", *job.OutputID, OutputProcessing).
					Updates(map[string]any{"status": OutputQueued, "started_at": nil}).
					Error; err != nil {
					return err
				}
			}
			requeued = true
			return nil
		}

		summary := "Worker 中断且 Provider 请求结果不确定，任务不会自动重试"
		if err := tx.Model(&model.ProviderAttempt{}).
			Where("job_id = ? AND status = ?", job.ID, "processing").
			Updates(map[string]any{
				"status": "unknown", "error_code": "provider_outcome_unknown",
				"error_summary": summary, "completed_at": &now,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&job).Updates(map[string]any{
			"locked_by": workerID, "heartbeat_at": &now, "last_error": summary,
		}).Error; err != nil {
			return err
		}
		job.LockedBy = workerID
		job.HeartbeatAt = &now
		uncertain = &job
		return nil
	})
	if err != nil {
		return false, err
	}
	if uncertain != nil {
		return true, p.fail(
			ctx,
			*uncertain,
			errors.New("provider request outcome is unknown after worker interruption"),
		)
	}
	return requeued, nil
}

func (p *Processor) claim(ctx context.Context, workerID string) (*model.GenerationJob, error) {
	var claimed model.GenerationJob
	found := false
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job model.GenerationJob
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND available_at <= ?", JobQueued, time.Now().UTC()).
			Order("created_at ASC").
			First(&job).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&job).Updates(map[string]any{
			"status":       JobProcessing,
			"locked_by":    workerID,
			"locked_at":    &now,
			"heartbeat_at": &now,
			"started_at":   &now,
			"attempts":     gorm.Expr("attempts + 1"),
			"version":      gorm.Expr("version + 1"),
		}).Error; err != nil {
			return err
		}
		if job.OutputID != nil {
			if err := tx.Model(&model.GenerationOutput{}).Where("id = ?", *job.OutputID).
				Updates(map[string]any{"status": OutputProcessing, "started_at": &now}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.GenerationTask{}).
			Where("id = ? AND status = ?", job.TaskID, StatusQueued).
			Updates(map[string]any{
				"status": StatusProcessing, "started_at": &now, "version": gorm.Expr("version + 1"),
			}).Error; err != nil {
			return err
		}
		job.Status = JobProcessing
		job.LockedBy = workerID
		job.LockedAt = &now
		claimed = job
		found = true
		return nil
	})
	if err != nil || !found {
		return nil, err
	}
	return &claimed, nil
}

func (p *Processor) execute(ctx context.Context, job model.GenerationJob) error {
	var task model.GenerationTask
	if err := p.db.WithContext(ctx).First(&task, "id = ?", job.TaskID).Error; err != nil {
		return err
	}
	if job.OutputID == nil {
		return errors.New("generation job has no output")
	}
	var promptVersion model.PromptVersion
	if err := p.db.WithContext(ctx).First(&promptVersion, "id = ?", task.PromptVersionID).Error; err != nil {
		return err
	}
	var sections prompt.Sections
	var settings Settings
	if err := json.Unmarshal(promptVersion.Sections, &sections); err != nil {
		return err
	}
	if err := json.Unmarshal(task.Settings, &settings); err != nil {
		return err
	}
	adapter, err := p.factory.FromSnapshot(task.ProviderBaseURLSnapshot, task.APIKeyCiphertextSnapshot)
	if err != nil {
		return err
	}
	attempt := model.ProviderAttempt{
		JobID:      job.ID,
		ProviderID: task.ProviderID,
		ModelID:    task.ModelID,
		AttemptNo:  max(1, job.Attempts+1),
		Status:     "processing",
		StartedAt:  time.Now().UTC(),
	}
	if err := p.db.WithContext(ctx).Create(&attempt).Error; err != nil {
		return err
	}

	fullPrompt := generationProviderPrompt(
		prompt.BuildGenerationPrompt(promptVersion.Source, sections),
		task.Mode,
		settings.ReferenceStrength,
	)
	started := time.Now()
	p.logger.Info(
		"generation_provider_call_started",
		"job_id", job.ID,
		"task_id", task.ID,
		"output_id", job.OutputID,
		"provider_id", task.ProviderID,
		"model_id", task.ModelID,
		"mode", task.Mode,
	)
	var images []provider.GeneratedImage
	if task.Mode == "text-to-image" {
		images, err = adapter.GenerateTextToImage(ctx, provider.TextToImageRequest{
			Model: task.ModelNameSnapshot, Prompt: fullPrompt, OutputCount: 1,
			Size: SizeForAspectRatio(settings.AspectRatio), UserReference: task.UserID,
		})
	} else {
		inputs, closers, openErr := p.imageInputs(ctx, task.ID)
		if openErr != nil {
			err = openErr
		} else {
			defer closeAll(closers)
			images, err = adapter.GenerateImageToImage(ctx, provider.ImageToImageRequest{
				Model: task.ModelNameSnapshot, Prompt: fullPrompt, Images: inputs,
				OutputCount: 1, Size: SizeForAspectRatio(settings.AspectRatio), UserReference: task.UserID,
			})
		}
	}
	if err != nil {
		latency := time.Since(started)
		p.logger.Warn(
			"generation_provider_call_failed",
			"job_id", job.ID,
			"task_id", task.ID,
			"provider_id", task.ProviderID,
			"model_id", task.ModelID,
			"mode", task.Mode,
			"latency_ms", latency.Milliseconds(),
			"error_kind", providerErrorKind(err),
		)
		now := time.Now().UTC()
		_ = p.db.WithContext(ctx).Model(&attempt).Updates(map[string]any{
			"status": "failed", "latency_millis": latency.Milliseconds(),
			"error_code": "provider_error", "error_summary": safeGenerationError(err), "completed_at": &now,
		}).Error
		return err
	}
	if len(images) != 1 {
		return fmt.Errorf("provider returned %d images, want 1", len(images))
	}
	p.logger.Info(
		"generation_provider_call_succeeded",
		"job_id", job.ID,
		"task_id", task.ID,
		"provider_id", task.ProviderID,
		"model_id", task.ModelID,
		"mode", task.Mode,
		"latency_ms", time.Since(started).Milliseconds(),
		"image_count", len(images),
	)
	image := images[0]
	filename := fmt.Sprintf("result-%s.png", job.ID)
	created, err := p.assets.CreateGenerated(ctx, task.UserID, filename, image.ContentType, image.Data)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	accepted := true
	_ = p.db.WithContext(ctx).Model(&attempt).Updates(map[string]any{
		"status": "succeeded", "latency_millis": time.Since(started).Milliseconds(),
		"external_request_id": image.RequestID, "request_accepted": &accepted, "completed_at": &now,
	}).Error
	return p.succeed(ctx, job, *created, image.RequestID)
}

func generationProviderPrompt(basePrompt, mode string, referenceStrength int) string {
	if mode != "image-to-image" {
		return basePrompt
	}
	return fmt.Sprintf(
		"%s\n参考素材影响强度：%d/100。强度越高越贴近原图与参考图，越低越允许模型自由发挥。",
		basePrompt,
		referenceStrength,
	)
}

func providerErrorKind(err error) string {
	var providerErr *provider.Error
	if errors.As(err, &providerErr) {
		return string(providerErr.Kind)
	}
	return "internal"
}

func (p *Processor) succeed(
	ctx context.Context,
	job model.GenerationJob,
	resultAsset model.Asset,
	requestID string,
) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.GenerationTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, "id = ?", job.TaskID).Error; err != nil {
			return err
		}
		var output model.GenerationOutput
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&output, "id = ?", *job.OutputID).Error; err != nil {
			return err
		}
		if output.Status == OutputSucceeded {
			return nil
		}
		if err := p.credits.SettleTx(tx, task.CreditReservationID, 1); err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&output).Updates(map[string]any{
			"status": OutputSucceeded, "asset_id": resultAsset.ID,
			"provider_response_id": requestID, "completed_at": &now, "version": gorm.Expr("version + 1"),
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.AssetRelation{
			AssetID: resultAsset.ID, ResourceType: "generation_task",
			ResourceID: task.ID, RelationType: "result",
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.GenerationJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status": JobCompleted, "completed_at": &now, "heartbeat_at": &now,
		}).Error; err != nil {
			return err
		}
		completed := task.CompletedOutputs + 1
		status, finishedAt := terminalState(completed, task.FailedOutputs, task.OutputCount, now)
		updates := map[string]any{
			"completed_outputs": completed,
			"spent_credits":     completed,
			"status":            status,
			"version":           gorm.Expr("version + 1"),
		}
		if finishedAt != nil {
			updates["completed_at"] = finishedAt
		}
		return tx.Model(&task).Updates(updates).Error
	})
}

func (p *Processor) fail(ctx context.Context, job model.GenerationJob, cause error) error {
	if job.OutputID == nil {
		return cause
	}
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.GenerationTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, "id = ?", job.TaskID).Error; err != nil {
			return err
		}
		var output model.GenerationOutput
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&output, "id = ?", *job.OutputID).Error; err != nil {
			return err
		}
		if output.Status == OutputFailed {
			return nil
		}
		if _, err := p.credits.ReleaseTx(tx, task.CreditReservationID, 1, "AI 输出失败释放次数"); err != nil {
			return err
		}
		now := time.Now().UTC()
		summary := safeGenerationError(cause)
		if err := tx.Model(&output).Updates(map[string]any{
			"status": OutputFailed, "error_code": "provider_failed",
			"error_summary": summary, "completed_at": &now, "version": gorm.Expr("version + 1"),
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.GenerationJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status": JobFailed, "last_error": summary, "completed_at": &now, "heartbeat_at": &now,
		}).Error; err != nil {
			return err
		}
		failed := task.FailedOutputs + 1
		refunded := task.RefundedCredits + 1
		status, finishedAt := terminalState(task.CompletedOutputs, failed, task.OutputCount, now)
		updates := map[string]any{
			"failed_outputs":   failed,
			"refunded_credits": refunded,
			"status":           status,
			"error_code":       "provider_failed",
			"error_summary":    summary,
			"version":          gorm.Expr("version + 1"),
		}
		if finishedAt != nil {
			updates["completed_at"] = finishedAt
		}
		return tx.Model(&task).Updates(updates).Error
	})
}

func (p *Processor) imageInputs(
	ctx context.Context,
	taskID string,
) ([]provider.ImageInput, []io.Closer, error) {
	var links []model.GenerationTaskAsset
	if err := p.db.WithContext(ctx).Where("task_id = ?", taskID).
		Order("CASE WHEN usage = 'source' THEN 0 ELSE 1 END ASC").
		Order("created_at ASC").Find(&links).Error; err != nil {
		return nil, nil, err
	}
	inputs := make([]provider.ImageInput, 0, len(links))
	closers := make([]io.Closer, 0, len(links))
	for _, link := range links {
		if link.Usage != asset.KindSource && link.Usage != asset.KindReference {
			continue
		}
		assetModel, err := p.assets.GetByID(ctx, link.AssetID)
		if err != nil {
			closeAll(closers)
			return nil, nil, err
		}
		reader, err := p.assets.Open(ctx, *assetModel)
		if err != nil {
			closeAll(closers)
			return nil, nil, err
		}
		closers = append(closers, reader)
		inputs = append(inputs, provider.ImageInput{
			Filename: assetModel.OriginalName, ContentType: assetModel.MIMEType, Reader: reader,
		})
	}
	if len(inputs) == 0 {
		return nil, nil, errors.New("image-to-image task has no input image")
	}
	return inputs, closers, nil
}

func terminalState(completed, failed, total int, now time.Time) (string, *time.Time) {
	if completed+failed < total {
		return StatusProcessing, nil
	}
	switch {
	case completed == total:
		return StatusCompleted, &now
	case completed > 0:
		return StatusPartial, &now
	default:
		return StatusFailed, &now
	}
}

func closeAll(closers []io.Closer) {
	for _, closer := range closers {
		_ = closer.Close()
	}
}
