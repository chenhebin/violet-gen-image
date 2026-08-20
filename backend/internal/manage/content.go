package manage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/generation"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/prompt"
	"yingyan.local/backend/internal/retouch"
)

type TaskQuery struct {
	Page             int
	PageSize         int
	Search           string
	Status           string
	Mode             string
	UserID           string
	ProviderID       string
	ModelID          string
	HasRetouchTicket *bool
}

type AssetQuery struct {
	Page     int
	PageSize int
	Search   string
	Kind     string
	UserID   string
	TaskID   string
	TicketID string
	Retained *bool
}

type RetouchQuery struct {
	Page     int
	PageSize int
	Search   string
	Status   string
	UserID   string
	Pending  bool
	SLA      string
}

type AuditQuery struct {
	Page         int
	PageSize     int
	Search       string
	OperatorID   string
	Action       string
	ResourceType string
	Result       string
	StartAt      *time.Time
	EndAt        *time.Time
}

func (s *Service) ListTasks(ctx context.Context, input TaskQuery) (PageResult[ManagedTaskSummaryDTO], error) {
	input.Page, input.PageSize = pageValues(input.Page, input.PageSize)
	query := s.db.WithContext(ctx).Model(&model.GenerationTask{}).
		Joins("LEFT JOIN users AS task_owner ON task_owner.id = generation_tasks.user_id")
	if input.Search != "" {
		search := "%" + strings.TrimSpace(input.Search) + "%"
		query = query.Where(
			"generation_tasks.title ILIKE ? OR CAST(generation_tasks.id AS text) ILIKE ? "+
				"OR task_owner.email ILIKE ?",
			search,
			search,
			search,
		)
	}
	if input.Status != "" {
		query = query.Where("generation_tasks.status = ?", input.Status)
	}
	if input.Mode != "" {
		query = query.Where("generation_tasks.mode = ?", input.Mode)
	}
	if input.UserID != "" {
		query = query.Where("generation_tasks.user_id = ?", input.UserID)
	}
	if input.ProviderID != "" {
		query = query.Where("generation_tasks.provider_id = ?", input.ProviderID)
	}
	if input.ModelID != "" {
		query = query.Where("generation_tasks.model_id = ?", input.ModelID)
	}
	if input.HasRetouchTicket != nil {
		condition := "EXISTS"
		if !*input.HasRetouchTicket {
			condition = "NOT EXISTS"
		}
		query = query.Where(condition + " (SELECT 1 FROM retouch_tickets rt WHERE rt.task_id = generation_tasks.id)")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PageResult[ManagedTaskSummaryDTO]{}, err
	}
	var tasks []model.GenerationTask
	if err := query.Select("generation_tasks.*").
		Order("generation_tasks.created_at DESC").
		Offset((input.Page - 1) * input.PageSize).
		Limit(input.PageSize).
		Find(&tasks).Error; err != nil {
		return PageResult[ManagedTaskSummaryDTO]{}, err
	}
	items := make([]ManagedTaskSummaryDTO, 0, len(tasks))
	for _, task := range tasks {
		value, err := s.taskSummary(ctx, task)
		if err != nil {
			return PageResult[ManagedTaskSummaryDTO]{}, err
		}
		items = append(items, value)
	}
	return PageResult[ManagedTaskSummaryDTO]{
		Items: items, Page: input.Page, PageSize: input.PageSize,
		Total: total, HasMore: int64(input.Page*input.PageSize) < total,
	}, nil
}

func (s *Service) GetTask(ctx context.Context, taskID string) (*ManagedTaskDTO, error) {
	var task model.GenerationTask
	if err := s.db.WithContext(ctx).First(&task, "id = ?", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierror.New(http.StatusNotFound, apierror.CodeTaskNotFound, "任务不存在", nil)
		}
		return nil, err
	}
	summary, err := s.taskSummary(ctx, task)
	if err != nil {
		return nil, err
	}
	var confirmed model.PromptVersion
	if err := s.db.WithContext(ctx).First(&confirmed, "id = ?", task.PromptVersionID).Error; err != nil {
		return nil, err
	}
	var confirmedSections prompt.Sections
	if err := json.Unmarshal(confirmed.Sections, &confirmedSections); err != nil {
		return nil, err
	}
	optimizedSections := confirmedSections
	if confirmed.ProviderID != nil {
		var draft model.PromptVersion
		if err := s.db.WithContext(ctx).
			Where("user_id = ? AND source = ? AND status = ? AND created_at <= ?", task.UserID, confirmed.Source, prompt.StatusDraft, confirmed.CreatedAt).
			Order("created_at DESC").First(&draft).Error; err == nil {
			_ = json.Unmarshal(draft.Sections, &optimizedSections)
		}
	}
	var settings map[string]any
	if err := json.Unmarshal(task.Settings, &settings); err != nil {
		return nil, err
	}
	var taskLinks []model.GenerationTaskAsset
	if err := s.db.WithContext(ctx).Where("task_id = ?", task.ID).Find(&taskLinks).Error; err != nil {
		return nil, err
	}
	assets := make([]ManagedAssetDTO, 0, len(taskLinks))
	for _, link := range taskLinks {
		value, err := s.GetAsset(ctx, link.AssetID)
		if err != nil {
			return nil, err
		}
		value.TaskID = task.ID
		if link.Usage == asset.KindReference {
			value.Role = link.ReferenceRole
		}
		assets = append(assets, *value)
	}
	var outputs []model.GenerationOutput
	if err := s.db.WithContext(ctx).Where("task_id = ? AND status = ?", task.ID, generation.OutputSucceeded).
		Order("output_index ASC").Find(&outputs).Error; err != nil {
		return nil, err
	}
	results := make([]ManagedAssetDTO, 0, len(outputs))
	for _, output := range outputs {
		if output.AssetID == nil {
			continue
		}
		value, err := s.GetAsset(ctx, *output.AssetID)
		if err != nil {
			return nil, err
		}
		value.TaskID = task.ID
		results = append(results, *value)
	}
	var jobs []model.GenerationJob
	if err := s.db.WithContext(ctx).Where("task_id = ?", task.ID).Order("created_at ASC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	jobIDs := make([]string, 0, len(jobs))
	for _, job := range jobs {
		jobIDs = append(jobIDs, job.ID)
	}
	attempts := make([]model.ProviderAttempt, 0)
	if len(jobIDs) > 0 {
		if err := s.db.WithContext(ctx).Where("job_id IN ?", jobIDs).Order("started_at ASC").Find(&attempts).Error; err != nil {
			return nil, err
		}
	}
	providerAttempts := make([]ProviderAttemptDTO, 0, len(attempts))
	for _, attempt := range attempts {
		providerAttempts = append(providerAttempts, providerAttemptDTO(attempt))
	}
	value := &ManagedTaskDTO{
		ManagedTaskSummaryDTO: summary,
		SourceRequirement:     confirmed.Source,
		OptimizedPrompt:       optimizedSections,
		ConfirmedPrompt:       confirmedSections,
		Settings:              settings,
		Assets:                assets,
		Results:               results,
		ProviderAttempts:      providerAttempts,
		ExecutionSnapshot: map[string]any{
			"providerId": task.ProviderID, "providerName": task.ProviderNameSnapshot,
			"modelId": task.ModelNameSnapshot, "modelName": task.ModelDisplayNameSnapshot,
			"configVersion": task.ModelConfigVersion,
		},
		ErrorMessage: task.ErrorSummary,
	}
	var ticket model.RetouchTicket
	if err := s.db.WithContext(ctx).Where("task_id = ?", task.ID).Order("created_at DESC").First(&ticket).Error; err == nil {
		manageTicket, ticketErr := s.retouches.GetManage(ctx, ticket.ID)
		if ticketErr == nil {
			summary := retouch.SummaryDTO{
				ID: manageTicket.ID, TicketNo: manageTicket.TicketNo,
				TaskID: manageTicket.TaskID, TaskTitle: manageTicket.TaskTitle,
				Status: manageTicket.Status, CreatedAt: manageTicket.CreatedAt, UpdatedAt: manageTicket.UpdatedAt,
			}
			if manageTicket.Quote != nil {
				credits := manageTicket.Quote.Credits
				summary.QuoteCredits = &credits
			}
			value.RetouchTicket = &summary
		}
	}
	return value, nil
}

func providerAttemptDTO(value model.ProviderAttempt) ProviderAttemptDTO {
	return ProviderAttemptDTO{
		ID: value.ID, JobID: value.JobID, AttemptNo: value.AttemptNo,
		Operation: value.Operation, Method: value.HTTPMethod, Path: value.EndpointPath,
		Model: value.ModelName, Status: value.Status,
		ExternalRequestID: value.ExternalRequestID, ResponseStatus: value.ResponseStatus,
		LatencyMillis: value.LatencyMillis, ErrorCode: value.ErrorCode,
		ErrorKind: value.ErrorKind, ErrorSummary: value.ErrorSummary,
		RequestSummary:   decodeJSONMap(value.RequestSummary),
		ResponseMetadata: decodeJSONMap(value.ResponseMetadata),
		StartedAt:        value.StartedAt, CompletedAt: value.CompletedAt,
	}
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func (s *Service) ListAssets(ctx context.Context, input AssetQuery) (PageResult[ManagedAssetDTO], error) {
	input.Page, input.PageSize = pageValues(input.Page, input.PageSize)
	query := s.db.WithContext(ctx).Model(&model.Asset{}).
		Joins("LEFT JOIN users AS asset_owner ON asset_owner.id = assets.owner_user_id")
	if input.Search != "" {
		search := "%" + strings.TrimSpace(input.Search) + "%"
		query = query.Where(
			"assets.original_name ILIKE ? OR CAST(assets.id AS text) ILIKE ? OR asset_owner.email ILIKE ?",
			search,
			search,
			search,
		)
	}
	if input.Kind != "" {
		query = query.Where("assets.kind = ?", input.Kind)
	}
	if input.UserID != "" {
		query = query.Where("assets.owner_user_id = ?", input.UserID)
	}
	if input.Retained != nil {
		query = query.Where("assets.retain_permanently = ?", *input.Retained)
	}
	if input.TaskID != "" {
		query = query.Where("EXISTS (SELECT 1 FROM asset_relations ar WHERE ar.asset_id = assets.id AND ar.resource_type = 'generation_task' AND ar.resource_id = ?)", input.TaskID)
	}
	if input.TicketID != "" {
		query = query.Where("EXISTS (SELECT 1 FROM asset_relations ar WHERE ar.asset_id = assets.id AND ar.resource_type = 'retouch_ticket' AND ar.resource_id = ?)", input.TicketID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PageResult[ManagedAssetDTO]{}, err
	}
	var assets []model.Asset
	if err := query.Select("assets.*").Order("assets.created_at DESC").
		Offset((input.Page - 1) * input.PageSize).
		Limit(input.PageSize).
		Find(&assets).Error; err != nil {
		return PageResult[ManagedAssetDTO]{}, err
	}
	items := make([]ManagedAssetDTO, 0, len(assets))
	for _, assetModel := range assets {
		value, err := s.managedAssetDTO(ctx, assetModel)
		if err != nil {
			return PageResult[ManagedAssetDTO]{}, err
		}
		items = append(items, value)
	}
	return PageResult[ManagedAssetDTO]{
		Items: items, Page: input.Page, PageSize: input.PageSize,
		Total: total, HasMore: int64(input.Page*input.PageSize) < total,
	}, nil
}

func (s *Service) GetAsset(ctx context.Context, assetID string) (*ManagedAssetDTO, error) {
	assetModel, err := s.assets.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	value, err := s.managedAssetDTO(ctx, *assetModel)
	return &value, err
}

func (s *Service) ListRetouch(ctx context.Context, input RetouchQuery) (PageResult[retouch.ManageSummaryDTO], error) {
	input.Page, input.PageSize = pageValues(input.Page, input.PageSize)
	query := s.db.WithContext(ctx).Model(&model.RetouchTicket{}).
		Joins("LEFT JOIN generation_tasks AS retouch_task ON retouch_task.id = retouch_tickets.task_id").
		Joins("LEFT JOIN users AS retouch_user ON retouch_user.id = retouch_tickets.user_id")
	if input.Status != "" {
		query = query.Where("retouch_tickets.status = ?", input.Status)
	}
	if input.Pending {
		query = query.Where("retouch_tickets.status NOT IN ?", []string{
			retouch.StatusDelivered,
			retouch.StatusRejected,
			retouch.StatusCancelled,
		})
	}
	if input.SLA == "overdue" || input.SLA == "due-soon" {
		now := time.Now().UTC()
		dueExpr := "CASE WHEN retouch_tickets.status IN ('submitted','quote_pending') THEN retouch_tickets.quote_due_at WHEN retouch_tickets.revision_used = true AND retouch_tickets.revision_due_at IS NOT NULL THEN retouch_tickets.revision_due_at ELSE retouch_tickets.first_delivery_due_at END"
		query = query.Where("("+dueExpr+") IS NOT NULL")
		if input.SLA == "overdue" {
			query = query.Where("("+dueExpr+") <= ?", now)
		} else {
			query = query.Where("("+dueExpr+") > ? AND ("+dueExpr+") <= ?", now, now.Add(24*time.Hour))
		}
	}
	if input.UserID != "" {
		query = query.Where("retouch_tickets.user_id = ?", input.UserID)
	}
	if input.Search != "" {
		search := "%" + strings.TrimSpace(input.Search) + "%"
		query = query.Where(
			"retouch_tickets.ticket_no ILIKE ? OR retouch_tickets.requirements ILIKE ? "+
				"OR retouch_task.title ILIKE ? OR retouch_user.email ILIKE ?",
			search,
			search,
			search,
			search,
		)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PageResult[retouch.ManageSummaryDTO]{}, err
	}
	var tickets []model.RetouchTicket
	if err := query.Select("retouch_tickets.*").
		Order("retouch_tickets.created_at DESC").
		Offset((input.Page - 1) * input.PageSize).
		Limit(input.PageSize).
		Find(&tickets).Error; err != nil {
		return PageResult[retouch.ManageSummaryDTO]{}, err
	}
	items := make([]retouch.ManageSummaryDTO, 0, len(tickets))
	for _, ticket := range tickets {
		detail, err := s.retouches.GetManage(ctx, ticket.ID)
		if err != nil {
			return PageResult[retouch.ManageSummaryDTO]{}, err
		}
		summary := retouch.SummaryDTO{
			ID: detail.ID, TicketNo: detail.TicketNo, TaskID: detail.TaskID,
			TaskTitle: detail.TaskTitle, Status: detail.Status,
			CreatedAt: detail.CreatedAt, UpdatedAt: detail.UpdatedAt,
		}
		if detail.Quote != nil {
			credits := detail.Quote.Credits
			summary.QuoteCredits = &credits
		}
		summary.SLA = detail.SLA
		items = append(items, retouch.ManageSummaryDTO{SummaryDTO: summary, User: detail.User})
	}
	return PageResult[retouch.ManageSummaryDTO]{
		Items: items, Page: input.Page, PageSize: input.PageSize,
		Total: total, HasMore: int64(input.Page*input.PageSize) < total,
	}, nil
}

func (s *Service) ListAudits(ctx context.Context, input AuditQuery) (PageResult[AuditDTO], error) {
	input.Page, input.PageSize = pageValues(input.Page, input.PageSize)
	query := s.db.WithContext(ctx).Model(&model.AuditLog{})
	if input.Search != "" {
		search := "%" + strings.TrimSpace(input.Search) + "%"
		query = query.Where(
			"admin_email ILIKE ? OR action ILIKE ? OR resource_type ILIKE ? OR CAST(resource_id AS text) ILIKE ? OR request_id ILIKE ?",
			search, search, search, search, search,
		)
	}
	if input.OperatorID != "" {
		query = query.Where("admin_id = ?", input.OperatorID)
	}
	if input.Action != "" {
		query = query.Where("action = ?", input.Action)
	}
	if input.ResourceType != "" {
		query = query.Where("resource_type = ?", input.ResourceType)
	}
	if input.Result != "" {
		query = query.Where("result = ?", input.Result)
	}
	if input.StartAt != nil {
		query = query.Where("created_at >= ?", *input.StartAt)
	}
	if input.EndAt != nil {
		query = query.Where("created_at <= ?", *input.EndAt)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PageResult[AuditDTO]{}, err
	}
	var logs []model.AuditLog
	if err := query.Order("created_at DESC").Offset((input.Page - 1) * input.PageSize).
		Limit(input.PageSize).Find(&logs).Error; err != nil {
		return PageResult[AuditDTO]{}, err
	}
	items := make([]AuditDTO, 0, len(logs))
	for _, log := range logs {
		before := map[string]any{}
		after := map[string]any{}
		_ = json.Unmarshal(log.BeforeSummary, &before)
		_ = json.Unmarshal(log.AfterSummary, &after)
		operatorID := ""
		resourceID := ""
		if log.AdminID != nil {
			operatorID = *log.AdminID
		}
		if log.ResourceID != nil {
			resourceID = *log.ResourceID
		}
		items = append(items, AuditDTO{
			ID: log.ID, OperatorID: operatorID, OperatorEmail: log.AdminEmail,
			OperatorRole: log.AdminRole, Action: log.Action, ResourceType: log.ResourceType,
			ResourceID: resourceID, Before: before, After: after, Reason: log.Reason,
			Result: log.Result, RequestID: log.RequestID, CreatedAt: log.CreatedAt,
		})
	}
	return PageResult[AuditDTO]{
		Items: items, Page: input.Page, PageSize: input.PageSize,
		Total: total, HasMore: int64(input.Page*input.PageSize) < total,
	}, nil
}

func (s *Service) taskSummary(ctx context.Context, task model.GenerationTask) (ManagedTaskSummaryDTO, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", task.UserID).Error; err != nil {
		return ManagedTaskSummaryDTO{}, err
	}
	var tickets int64
	if err := s.db.WithContext(ctx).Model(&model.RetouchTicket{}).Where("task_id = ?", task.ID).Count(&tickets).Error; err != nil {
		return ManagedTaskSummaryDTO{}, err
	}
	progress := generation.EstimatedProgress(task, time.Now().UTC())
	return ManagedTaskSummaryDTO{
		ID: task.ID, OwnerID: task.UserID, OwnerEmail: user.Email,
		Title: task.Title, Mode: task.Mode, Status: task.Status, Progress: progress,
		RequestedCount: task.OutputCount, SuccessfulCount: task.CompletedOutputs,
		ReservedCredits: task.ReservedCredits, SpentCredits: task.SpentCredits,
		RefundedCredits: task.RefundedCredits, ProviderName: task.ProviderNameSnapshot,
		ModelName: task.ModelDisplayNameSnapshot, HasRetouchTicket: tickets > 0,
		CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
	}, nil
}

func (s *Service) managedAssetDTO(ctx context.Context, value model.Asset) (ManagedAssetDTO, error) {
	dto, err := s.assets.DTO(ctx, value)
	if err != nil && value.CleanedAt == nil {
		return ManagedAssetDTO{}, err
	}
	ownerEmail := ""
	ownerID := ""
	if value.OwnerUserID != nil {
		ownerID = *value.OwnerUserID
		var user model.User
		if s.db.WithContext(ctx).First(&user, "id = ?", ownerID).Error == nil {
			ownerEmail = user.Email
		}
	}
	var taskID, ticketID string
	var relations []model.AssetRelation
	if err := s.db.WithContext(ctx).Where("asset_id = ?", value.ID).Find(&relations).Error; err != nil {
		return ManagedAssetDTO{}, err
	}
	for _, relation := range relations {
		if relation.ResourceType == "generation_task" {
			taskID = relation.ResourceID
		}
		if relation.ResourceType == "retouch_ticket" {
			ticketID = relation.ResourceID
		}
	}
	retention := value.CreatedAt.Add(90 * 24 * time.Hour)
	return ManagedAssetDTO{
		ID: value.ID, OwnerID: ownerID, OwnerEmail: ownerEmail, Name: value.OriginalName,
		Kind: value.Kind, Role: value.ReferenceRole, MIMEType: value.MIMEType,
		Size: value.SizeBytes, Width: value.Width, Height: value.Height,
		PreviewURL: dto.PreviewURL, PreviewURLExpiresAt: dto.PreviewURLExpiresAt,
		TaskID: taskID, TicketID: ticketID,
		Retained: value.RetainPermanently, RetentionExpiresAt: &retention,
		DeletedAt: value.CleanedAt, CreatedAt: value.CreatedAt,
	}, nil
}

var _ = generation.StatusQueued
