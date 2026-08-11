package retouch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/generation"
	"yingyan.local/backend/internal/idempotency"
	"yingyan.local/backend/internal/model"
)

const (
	StatusSubmitted            = "submitted"
	StatusQuotePending         = "quote_pending"
	StatusAccepted             = "accepted"
	StatusProcessing           = "processing"
	StatusAwaitingConfirmation = "awaiting_confirmation"
	StatusDelivered            = "delivered"
	StatusRejected             = "rejected"
	StatusCancelled            = "cancelled"
)

type Service struct {
	db          *gorm.DB
	credits     *credit.Service
	assets      *asset.Service
	generations *generation.Service
}

func New(
	db *gorm.DB,
	credits *credit.Service,
	assets *asset.Service,
	generations *generation.Service,
) *Service {
	return &Service{db: db, credits: credits, assets: assets, generations: generations}
}

func (s *Service) Create(
	ctx context.Context,
	userID string,
	taskID string,
	input CreateInput,
	key string,
) (*TicketDTO, error) {
	input.Requirement = strings.TrimSpace(input.Requirement)
	input.SelectedResultIDs = unique(input.SelectedResultIDs)
	input.SupplementalAssetIDs = unique(input.SupplementalAssetIDs)
	if len([]rune(input.Requirement)) < 1 || len([]rune(input.Requirement)) > 1000 {
		return nil, apierror.Invalid("人工修图需求长度需为 1 到 1000 字", nil)
	}
	if len(input.SelectedResultIDs) < 1 || len(input.SelectedResultIDs) > 4 {
		return nil, apierror.Invalid("请选择 1 到 4 张 AI 成片", nil)
	}
	if len(input.SupplementalAssetIDs) > 4 {
		return nil, apierror.Invalid("补充参考图不能超过 4 张", nil)
	}

	var ticketID string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "user", PrincipalID: userID, Method: http.MethodPost,
			Path: "/api/tasks/" + taskID + "/retouch-tickets", Key: key, Request: input,
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
			ticketID = reference.ID
			return nil
		}

		var task model.GenerationTask
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierror.New(http.StatusNotFound, apierror.CodeTaskNotFound, "任务不存在", nil)
		}
		if err != nil {
			return err
		}
		if task.Status != generation.StatusCompleted && task.Status != generation.StatusPartial {
			return apierror.New(http.StatusConflict, apierror.CodeRetouchIneligible, "该任务暂不满足人工修图条件", nil)
		}
		var active int64
		if err := tx.Model(&model.RetouchTicket{}).
			Where("task_id = ? AND status NOT IN ?", taskID, []string{StatusRejected, StatusCancelled}).
			Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return apierror.New(http.StatusConflict, apierror.CodeRetouchExists, "该任务已有人工修图工单", nil)
		}
		var outputs []model.GenerationOutput
		if err := tx.Where("task_id = ? AND status = ? AND asset_id IN ?", taskID, generation.OutputSucceeded, input.SelectedResultIDs).
			Find(&outputs).Error; err != nil {
			return err
		}
		if len(outputs) != len(input.SelectedResultIDs) {
			return apierror.New(http.StatusConflict, apierror.CodeRetouchIneligible, "所选成片不属于该任务", nil)
		}
		if len(input.SupplementalAssetIDs) > 0 {
			var supplementalCount int64
			if err := tx.Model(&model.Asset{}).
				Where("owner_user_id = ? AND kind = ? AND cleaned_at IS NULL AND id IN ?", userID, asset.KindRetouchReference, input.SupplementalAssetIDs).
				Count(&supplementalCount).Error; err != nil {
				return err
			}
			if supplementalCount != int64(len(input.SupplementalAssetIDs)) {
				return apierror.New(http.StatusNotFound, apierror.CodeAssetNotFound, "补充参考图不存在", nil)
			}
		}

		ticketID = uuid.NewString()
		ticket := model.RetouchTicket{
			BaseModel:    model.BaseModel{ID: ticketID},
			TicketNo:     ticketNumber(ticketID),
			UserID:       userID,
			TaskID:       taskID,
			Status:       StatusSubmitted,
			Requirements: input.Requirement,
			Version:      1,
		}
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		for _, resultID := range input.SelectedResultIDs {
			if err := tx.Create(&model.RetouchSelectedResult{TicketID: ticket.ID, AssetID: resultID}).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.AssetRelation{
				AssetID: resultID, ResourceType: "retouch_ticket", ResourceID: ticket.ID, RelationType: "selected_result",
			}).Error; err != nil {
				return err
			}
		}
		for _, assetID := range input.SupplementalAssetIDs {
			if err := tx.Create(&model.AssetRelation{
				AssetID: assetID, ResourceType: "retouch_ticket", ResourceID: ticket.ID, RelationType: "supplemental",
			}).Error; err != nil {
				return err
			}
		}
		if err := createEvent(tx, ticket, "", StatusSubmitted, "submit", "user", userID, "用户提交人工修图需求"); err != nil {
			return err
		}
		reference := struct {
			ID string `json:"id"`
		}{ID: ticket.ID}
		return idempotency.CompleteTx(tx, recordID, http.StatusCreated, 0, "retouch_ticket", &ticket.ID, reference)
	})
	if err != nil {
		return nil, err
	}
	return s.GetUser(ctx, userID, ticketID)
}

func (s *Service) Quote(
	ctx context.Context,
	adminID string,
	ticketID string,
	input QuoteInput,
	key string,
) (*ManageTicketDTO, error) {
	input.Note = strings.TrimSpace(input.Note)
	if input.Credits < 1 || input.Credits > 999 || len([]rune(input.Note)) > 500 {
		return nil, apierror.Invalid("报价需为 1 到 999 次，说明最多 500 字", nil)
	}
	err := s.mutate(ctx, "manage", adminID, "/api/manage/retouch-tickets/"+ticketID+"/quote", key, input, ticketID,
		func(tx *gorm.DB, ticket *model.RetouchTicket) error {
			if ticket.Status != StatusSubmitted && ticket.Status != StatusQuotePending {
				return stateError()
			}
			if ticket.CurrentQuoteID != nil {
				if err := tx.Model(&model.RetouchQuote{}).Where("id = ?", *ticket.CurrentQuoteID).
					Updates(map[string]any{"status": "invalidated", "invalidated_at": time.Now().UTC()}).Error; err != nil {
					return err
				}
			}
			var count int64
			if err := tx.Model(&model.RetouchQuote{}).Where("ticket_id = ?", ticket.ID).Count(&count).Error; err != nil {
				return err
			}
			quote := model.RetouchQuote{
				TicketID: ticket.ID, QuoteVersion: int(count) + 1, Credits: input.Credits,
				Notes: input.Note, Status: "active", CreatedBy: adminID,
			}
			if err := tx.Create(&quote).Error; err != nil {
				return err
			}
			now := time.Now().UTC()
			from := ticket.Status
			if err := tx.Model(ticket).Updates(map[string]any{
				"status": StatusQuotePending, "current_quote_id": quote.ID,
				"quoted_at": &now, "version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			return createEvent(tx, *ticket, from, StatusQuotePending, "quote", "manage", adminID, input.Note)
		})
	if err != nil {
		return nil, err
	}
	return s.GetManage(ctx, ticketID)
}

func (s *Service) AcceptQuote(
	ctx context.Context,
	userID string,
	ticketID string,
	quoteID string,
	key string,
) (*TicketDTO, int, error) {
	request := struct {
		QuoteID string `json:"quoteId"`
	}{QuoteID: quoteID}
	err := s.mutate(ctx, "user", userID, "/api/retouch-tickets/"+ticketID+"/quote/accept", key, request, ticketID,
		func(tx *gorm.DB, ticket *model.RetouchTicket) error {
			if ticket.UserID != userID || ticket.Status != StatusQuotePending ||
				ticket.CurrentQuoteID == nil || *ticket.CurrentQuoteID != quoteID {
				return apierror.New(http.StatusConflict, apierror.CodeRetouchQuote, "报价已失效，请刷新后重试", nil)
			}
			var quote model.RetouchQuote
			if err := tx.Where("id = ? AND status = ?", quoteID, "active").First(&quote).Error; err != nil {
				return apierror.New(http.StatusConflict, apierror.CodeRetouchQuote, "报价已失效", nil)
			}
			reservation, _, err := s.credits.ReserveTx(tx, userID, "retouch", ticket.ID, quote.Credits, "人工修图报价预占")
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			if err := tx.Model(&quote).Updates(map[string]any{"status": "accepted", "accepted_at": &now}).Error; err != nil {
				return err
			}
			if err := tx.Model(ticket).Updates(map[string]any{
				"status": StatusAccepted, "credit_reservation_id": reservation.ID,
				"reserved_credits": quote.Credits, "accepted_at": &now,
				"version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			return createEvent(tx, *ticket, StatusQuotePending, StatusAccepted, "accept_quote", "user", userID, "用户接受报价")
		})
	if err != nil {
		return nil, 0, err
	}
	ticket, err := s.GetUser(ctx, userID, ticketID)
	if err != nil {
		return nil, 0, err
	}
	balance, err := s.credits.Balance(userID)
	return ticket, balance, err
}

func (s *Service) Start(ctx context.Context, adminID, ticketID, key string) (*ManageTicketDTO, error) {
	err := s.mutate(ctx, "manage", adminID, "/api/manage/retouch-tickets/"+ticketID+"/start", key, struct{}{}, ticketID,
		func(tx *gorm.DB, ticket *model.RetouchTicket) error {
			if ticket.Status != StatusAccepted || ticket.CreditReservationID == nil {
				return stateError()
			}
			if err := s.credits.SettleTx(tx, *ticket.CreditReservationID, ticket.ReservedCredits); err != nil {
				return err
			}
			now := time.Now().UTC()
			if err := tx.Model(ticket).Updates(map[string]any{
				"status": StatusProcessing, "spent_credits": ticket.ReservedCredits,
				"started_at": &now, "version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			return createEvent(tx, *ticket, StatusAccepted, StatusProcessing, "start", "manage", adminID, "人工修图开始处理")
		})
	if err != nil {
		return nil, err
	}
	return s.GetManage(ctx, ticketID)
}

func (s *Service) Deliver(
	ctx context.Context,
	adminID string,
	ticketID string,
	resultAssets []model.Asset,
	request DeliveryRequest,
	key string,
) (*ManageTicketDTO, error) {
	request.Note = strings.TrimSpace(request.Note)
	if len(resultAssets) < 1 || len(resultAssets) > 4 ||
		request.FileDigest == "" || len([]rune(request.Note)) > 500 {
		return nil, apierror.Invalid("请上传 1 到 4 张人工成片，说明最多 500 字", nil)
	}
	err := s.mutate(ctx, "manage", adminID, "/api/manage/retouch-tickets/"+ticketID+"/deliver", key, request, ticketID,
		func(tx *gorm.DB, ticket *model.RetouchTicket) error {
			if ticket.Status != StatusProcessing {
				return stateError()
			}
			version := 1
			if ticket.RevisionUsed {
				version = 2
			}
			for _, result := range resultAssets {
				if result.OwnerUserID == nil || *result.OwnerUserID != ticket.UserID || result.Kind != asset.KindRetouchResult {
					return apierror.Invalid("人工成片归属无效", nil)
				}
				if err := tx.Create(&model.RetouchDeliverable{
					TicketID: ticket.ID, AssetID: result.ID, CreatedBy: adminID, VersionNo: version,
				}).Error; err != nil {
					return err
				}
				if err := tx.Create(&model.AssetRelation{
					AssetID: result.ID, ResourceType: "retouch_ticket", ResourceID: ticket.ID, RelationType: "deliverable",
				}).Error; err != nil {
					return err
				}
			}
			now := time.Now().UTC()
			if err := tx.Model(ticket).Updates(map[string]any{
				"status": StatusAwaitingConfirmation, "delivered_at": &now,
				"version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			return createEvent(
				tx,
				*ticket,
				StatusProcessing,
				StatusAwaitingConfirmation,
				"deliver",
				"manage",
				adminID,
				request.Note,
			)
		})
	if err != nil {
		return nil, err
	}
	return s.GetManage(ctx, ticketID)
}

func (s *Service) ReplayDelivery(
	ctx context.Context,
	adminID string,
	ticketID string,
	request DeliveryRequest,
	key string,
) (*ManageTicketDTO, bool, error) {
	replay, err := idempotency.Lookup(s.db.WithContext(ctx), idempotency.Scope{
		PrincipalRealm: "manage",
		PrincipalID:    adminID,
		Method:         http.MethodPost,
		Path:           "/api/manage/retouch-tickets/" + ticketID + "/deliver",
		Key:            key,
		Request:        request,
	})
	if err != nil || replay == nil {
		return nil, false, err
	}
	value, err := s.GetManage(ctx, ticketID)
	return value, true, err
}

func (s *Service) RequestRevision(ctx context.Context, userID, ticketID, message, key string) (*TicketDTO, error) {
	message = strings.TrimSpace(message)
	if len([]rune(message)) < 1 || len([]rune(message)) > 500 {
		return nil, apierror.Invalid("返修说明长度需为 1 到 500 字", nil)
	}
	request := struct {
		Message string `json:"message"`
	}{Message: message}
	err := s.mutate(ctx, "user", userID, "/api/retouch-tickets/"+ticketID+"/revisions", key, request, ticketID,
		func(tx *gorm.DB, ticket *model.RetouchTicket) error {
			if ticket.UserID != userID || ticket.Status != StatusAwaitingConfirmation {
				return stateError()
			}
			if ticket.RevisionUsed {
				return apierror.New(http.StatusConflict, apierror.CodeRetouchRevision, "返修机会已使用", nil)
			}
			if err := tx.Create(&model.RetouchRevision{
				TicketID: ticket.ID, Requirements: message, RequestedBy: userID,
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(ticket).Updates(map[string]any{
				"status": StatusProcessing, "revision_used": true, "version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			return createEvent(tx, *ticket, StatusAwaitingConfirmation, StatusProcessing, "revision", "user", userID, message)
		})
	if err != nil {
		return nil, err
	}
	return s.GetUser(ctx, userID, ticketID)
}

func (s *Service) Confirm(ctx context.Context, userID, ticketID, key string) (*TicketDTO, error) {
	err := s.mutate(ctx, "user", userID, "/api/retouch-tickets/"+ticketID+"/confirm", key, struct{}{}, ticketID,
		func(tx *gorm.DB, ticket *model.RetouchTicket) error {
			if ticket.UserID != userID || ticket.Status != StatusAwaitingConfirmation {
				return stateError()
			}
			now := time.Now().UTC()
			if err := tx.Model(ticket).Updates(map[string]any{
				"status": StatusDelivered, "closed_at": &now, "version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			return createEvent(tx, *ticket, StatusAwaitingConfirmation, StatusDelivered, "confirm", "user", userID, "用户确认人工成片")
		})
	if err != nil {
		return nil, err
	}
	return s.GetUser(ctx, userID, ticketID)
}

func (s *Service) Cancel(ctx context.Context, userID, ticketID, key string) (*TicketDTO, int, error) {
	err := s.mutate(ctx, "user", userID, "/api/retouch-tickets/"+ticketID+"/cancel", key, struct{}{}, ticketID,
		func(tx *gorm.DB, ticket *model.RetouchTicket) error {
			if ticket.UserID != userID ||
				(ticket.Status != StatusSubmitted && ticket.Status != StatusQuotePending && ticket.Status != StatusAccepted) {
				return stateError()
			}
			from := ticket.Status
			refunded := 0
			if ticket.Status == StatusAccepted && ticket.CreditReservationID != nil {
				if _, err := s.credits.ReleaseTx(tx, *ticket.CreditReservationID, ticket.ReservedCredits, "用户开工前取消人工修图"); err != nil {
					return err
				}
				refunded = ticket.ReservedCredits
			}
			now := time.Now().UTC()
			if err := tx.Model(ticket).Updates(map[string]any{
				"status": StatusCancelled, "refunded_credits": refunded,
				"closed_at": &now, "closure_type": "user_cancelled", "version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			return createEvent(tx, *ticket, from, StatusCancelled, "cancel", "user", userID, "用户取消人工修图")
		})
	if err != nil {
		return nil, 0, err
	}
	ticket, err := s.GetUser(ctx, userID, ticketID)
	if err != nil {
		return nil, 0, err
	}
	balance, err := s.credits.Balance(userID)
	return ticket, balance, err
}

func (s *Service) Reject(ctx context.Context, adminID, ticketID, reason, key string) (*ManageTicketDTO, error) {
	return s.closeByAdmin(ctx, adminID, ticketID, reason, key, false)
}

func (s *Service) Fail(ctx context.Context, adminID, ticketID, reason, key string) (*ManageTicketDTO, error) {
	return s.closeByAdmin(ctx, adminID, ticketID, reason, key, true)
}

func (s *Service) closeByAdmin(
	ctx context.Context,
	adminID, ticketID, reason, key string,
	failure bool,
) (*ManageTicketDTO, error) {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) < 1 || len([]rune(reason)) > 500 {
		return nil, apierror.Invalid("原因长度需为 1 到 500 字", nil)
	}
	action := "reject"
	path := "/api/manage/retouch-tickets/" + ticketID + "/reject"
	if failure {
		action = "fail"
		path = "/api/manage/retouch-tickets/" + ticketID + "/fail"
	}
	request := struct {
		Reason string `json:"reason"`
	}{Reason: reason}
	err := s.mutate(ctx, "manage", adminID, path, key, request, ticketID,
		func(tx *gorm.DB, ticket *model.RetouchTicket) error {
			from := ticket.Status
			if !failure && from != StatusSubmitted && from != StatusQuotePending {
				return stateError()
			}
			if failure && from != StatusAccepted && from != StatusProcessing && from != StatusAwaitingConfirmation {
				return stateError()
			}
			refunded := 0
			if failure && ticket.CreditReservationID != nil {
				if from == StatusAccepted {
					if _, err := s.credits.ReleaseTx(tx, *ticket.CreditReservationID, ticket.ReservedCredits, reason); err != nil {
						return err
					}
				} else {
					if _, err := s.credits.RefundTx(tx, *ticket.CreditReservationID, ticket.SpentCredits, reason); err != nil {
						return err
					}
				}
				refunded = ticket.ReservedCredits
			}
			now := time.Now().UTC()
			closureType := "operator_rejected"
			if failure {
				closureType = "fulfillment_failed"
			}
			if err := tx.Model(ticket).Updates(map[string]any{
				"status": StatusRejected, "closure_type": closureType,
				"closed_reason": reason, "refunded_credits": refunded,
				"closed_at": &now, "version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
			return createEvent(tx, *ticket, from, StatusRejected, action, "manage", adminID, reason)
		})
	if err != nil {
		return nil, err
	}
	return s.GetManage(ctx, ticketID)
}

func (s *Service) ListUser(ctx context.Context, userID string) ([]TicketDTO, error) {
	var tickets []model.RetouchTicket
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("updated_at DESC").Find(&tickets).Error; err != nil {
		return nil, err
	}
	result := make([]TicketDTO, 0, len(tickets))
	for _, ticket := range tickets {
		value, err := s.ticketDTO(ctx, ticket)
		if err != nil {
			return nil, err
		}
		result = append(result, *value)
	}
	return result, nil
}

func (s *Service) GetUser(ctx context.Context, userID, ticketID string) (*TicketDTO, error) {
	var ticket model.RetouchTicket
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", ticketID, userID).First(&ticket).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierror.New(http.StatusNotFound, apierror.CodeRetouchNotFound, "人工修图工单不存在", nil)
	}
	if err != nil {
		return nil, err
	}
	return s.ticketDTO(ctx, ticket)
}

func (s *Service) GetManage(ctx context.Context, ticketID string) (*ManageTicketDTO, error) {
	var ticket model.RetouchTicket
	if err := s.db.WithContext(ctx).First(&ticket, "id = ?", ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierror.New(http.StatusNotFound, apierror.CodeRetouchNotFound, "人工修图工单不存在", nil)
		}
		return nil, err
	}
	value, err := s.ticketDTO(ctx, ticket)
	if err != nil {
		return nil, err
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", ticket.UserID).Error; err != nil {
		return nil, err
	}
	var task model.GenerationTask
	if err := s.db.WithContext(ctx).First(&task, "id = ?", ticket.TaskID).Error; err != nil {
		return nil, err
	}
	var promptVersion model.PromptVersion
	if err := s.db.WithContext(ctx).First(&promptVersion, "id = ?", task.PromptVersionID).Error; err != nil {
		return nil, err
	}
	return &ManageTicketDTO{
		TicketDTO: *value,
		User:      UserSummary{ID: user.ID, Email: user.Email, Status: user.Status},
		SourceTaskDetail: ManageSourceTaskDTO{
			ID: task.ID, Title: task.Title, Mode: task.Mode, Status: task.Status,
			ModelName: task.ModelDisplayNameSnapshot, SourceRequirement: promptVersion.Source,
		},
	}, nil
}

func (s *Service) mutate(
	ctx context.Context,
	realm, principalID, path, key string,
	request any,
	ticketID string,
	action func(*gorm.DB, *model.RetouchTicket) error,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: realm, PrincipalID: principalID,
			Method: http.MethodPost, Path: path, Key: key, Request: request,
		})
		if err != nil {
			return err
		}
		if replay != nil {
			return nil
		}
		var ticket model.RetouchTicket
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&ticket, "id = ?", ticketID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierror.New(http.StatusNotFound, apierror.CodeRetouchNotFound, "人工修图工单不存在", nil)
		}
		if err != nil {
			return err
		}
		if err := action(tx, &ticket); err != nil {
			return err
		}
		reference := struct {
			ID string `json:"id"`
		}{ID: ticketID}
		return idempotency.CompleteTx(tx, recordID, http.StatusOK, 0, "retouch_ticket", &ticketID, reference)
	})
}

func (s *Service) ticketDTO(ctx context.Context, ticket model.RetouchTicket) (*TicketDTO, error) {
	var task model.GenerationTask
	if err := s.db.WithContext(ctx).First(&task, "id = ?", ticket.TaskID).Error; err != nil {
		return nil, err
	}
	var selected []model.RetouchSelectedResult
	if err := s.db.WithContext(ctx).Where("ticket_id = ?", ticket.ID).Find(&selected).Error; err != nil {
		return nil, err
	}
	selectedResults := make([]generation.ResultDTO, 0, len(selected))
	for _, item := range selected {
		assetModel, err := s.assets.GetByID(ctx, item.AssetID)
		if err != nil {
			return nil, err
		}
		dto, err := s.assets.DTO(ctx, *assetModel)
		if err != nil {
			return nil, err
		}
		downloadURL, err := s.assets.DownloadURL(ctx, *assetModel)
		if err != nil {
			return nil, err
		}
		selectedResults = append(selectedResults, generation.ResultDTO{
			ID: assetModel.ID, URL: dto.PreviewURL, DownloadURL: downloadURL,
			Width: assetModel.Width, Height: assetModel.Height,
		})
	}
	var relations []model.AssetRelation
	if err := s.db.WithContext(ctx).Where(
		"resource_type = ? AND resource_id = ? AND relation_type = ?",
		"retouch_ticket", ticket.ID, "supplemental",
	).Find(&relations).Error; err != nil {
		return nil, err
	}
	supplemental := make([]asset.DTO, 0, len(relations))
	for _, relation := range relations {
		assetModel, err := s.assets.GetByID(ctx, relation.AssetID)
		if err != nil {
			return nil, err
		}
		dto, err := s.assets.DTO(ctx, *assetModel)
		if err != nil {
			return nil, err
		}
		supplemental = append(supplemental, dto)
	}
	var quoteDTO *QuoteDTO
	if ticket.CurrentQuoteID != nil {
		var quote model.RetouchQuote
		if s.db.WithContext(ctx).First(&quote, "id = ?", *ticket.CurrentQuoteID).Error == nil {
			quoteDTO = &QuoteDTO{ID: quote.ID, Credits: quote.Credits, CreatedAt: quote.CreatedAt}
		}
	}
	var events []model.RetouchEvent
	if err := s.db.WithContext(ctx).Where("ticket_id = ?", ticket.ID).Order("created_at ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	timeline := make([]TimelineEntry, 0, len(events))
	for _, event := range events {
		timeline = append(timeline, TimelineEntry{
			Status: event.ToStatus, Action: event.Action, Note: event.Summary, CreatedAt: event.CreatedAt,
		})
	}
	var revisionDTO *RevisionDTO
	var revision model.RetouchRevision
	if err := s.db.WithContext(ctx).Where("ticket_id = ?", ticket.ID).First(&revision).Error; err == nil {
		revisionDTO = &RevisionDTO{Message: revision.Requirements, RequestedAt: revision.CreatedAt}
	}
	deliverables := make([]generation.ResultDTO, 0)
	if ticket.Status == StatusAwaitingConfirmation || ticket.Status == StatusDelivered {
		var deliverableModels []model.RetouchDeliverable
		if err := s.db.WithContext(ctx).Where("ticket_id = ?", ticket.ID).Order("version_no DESC, created_at ASC").Find(&deliverableModels).Error; err != nil {
			return nil, err
		}
		latestVersion := 0
		for _, item := range deliverableModels {
			if item.VersionNo > latestVersion {
				latestVersion = item.VersionNo
			}
		}
		for _, item := range deliverableModels {
			if item.VersionNo != latestVersion {
				continue
			}
			assetModel, err := s.assets.GetByID(ctx, item.AssetID)
			if err != nil {
				return nil, err
			}
			dto, err := s.assets.DTO(ctx, *assetModel)
			if err != nil {
				return nil, err
			}
			downloadURL, err := s.assets.DownloadURL(ctx, *assetModel)
			if err != nil {
				return nil, err
			}
			deliverables = append(deliverables, generation.ResultDTO{
				ID: assetModel.ID, URL: dto.PreviewURL, DownloadURL: downloadURL,
				Width: assetModel.Width, Height: assetModel.Height,
			})
		}
	}
	return &TicketDTO{
		ID: ticket.ID, TicketNo: ticket.TicketNo, TaskID: ticket.TaskID, TaskTitle: task.Title,
		Status: ticket.Status, SelectedResults: selectedResults, Requirement: ticket.Requirements,
		SupplementalAssets: supplemental, Quote: quoteDTO, Timeline: timeline,
		ReservedCredits: ticket.ReservedCredits, SpentCredits: ticket.SpentCredits,
		RefundedCredits: ticket.RefundedCredits, Revision: revisionDTO, Deliverables: deliverables,
		CreatedAt: ticket.CreatedAt, UpdatedAt: ticket.UpdatedAt,
	}, nil
}

func createEvent(
	tx *gorm.DB,
	ticket model.RetouchTicket,
	from, to, action, realm, actorID, summary string,
) error {
	return tx.Create(&model.RetouchEvent{
		TicketID: ticket.ID, FromStatus: from, ToStatus: to, Action: action,
		ActorRealm: realm, ActorID: actorID, Summary: strings.TrimSpace(summary),
	}).Error
}

func stateError() error {
	return apierror.New(http.StatusConflict, apierror.CodeRetouchState, "工单状态不允许当前操作", nil)
}

func ticketNumber(id string) string {
	value := strings.ToUpper(strings.ReplaceAll(id, "-", ""))
	if len(value) > 8 {
		value = value[:8]
	}
	return fmt.Sprintf("RT-%s-%s", time.Now().UTC().Format("20060102"), value)
}

func unique(values []string) []string {
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
	return result
}
