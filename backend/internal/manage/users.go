package manage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/idempotency"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/platform/security"
)

type UserQuery struct {
	Page       int
	PageSize   int
	Search     string
	Status     string
	MinBalance *int
	MaxBalance *int
	HasTasks   *bool
	HasTickets *bool
}

func (s *Service) ListUsers(ctx context.Context, input UserQuery) (PageResult[ManagedUserDTO], error) {
	input.Page, input.PageSize = pageValues(input.Page, input.PageSize)
	query := s.db.WithContext(ctx).Model(&model.User{}).
		Joins("LEFT JOIN credit_accounts ON credit_accounts.user_id = users.id")
	if input.Search != "" {
		search := "%" + strings.TrimSpace(input.Search) + "%"
		query = query.Where(
			"users.email ILIKE ? OR CAST(users.id AS text) ILIKE ?",
			search,
			search,
		)
	}
	if input.Status != "" {
		query = query.Where("users.status = ?", input.Status)
	}
	if input.MinBalance != nil {
		query = query.Where("credit_accounts.balance >= ?", *input.MinBalance)
	}
	if input.MaxBalance != nil {
		query = query.Where("credit_accounts.balance <= ?", *input.MaxBalance)
	}
	if input.HasTasks != nil {
		condition := "EXISTS"
		if !*input.HasTasks {
			condition = "NOT EXISTS"
		}
		query = query.Where(condition + " (SELECT 1 FROM generation_tasks gt WHERE gt.user_id = users.id)")
	}
	if input.HasTickets != nil {
		condition := "EXISTS"
		if !*input.HasTickets {
			condition = "NOT EXISTS"
		}
		query = query.Where(condition + " (SELECT 1 FROM retouch_tickets rt WHERE rt.user_id = users.id)")
	}
	var total int64
	if err := query.Distinct("users.id").Count(&total).Error; err != nil {
		return PageResult[ManagedUserDTO]{}, err
	}
	var users []model.User
	if err := query.Select("users.*").Order("users.created_at DESC").
		Offset((input.Page - 1) * input.PageSize).Limit(input.PageSize).
		Find(&users).Error; err != nil {
		return PageResult[ManagedUserDTO]{}, err
	}
	items := make([]ManagedUserDTO, 0, len(users))
	for _, user := range users {
		value, err := s.userDTO(ctx, user)
		if err != nil {
			return PageResult[ManagedUserDTO]{}, err
		}
		items = append(items, value)
	}
	return PageResult[ManagedUserDTO]{
		Items: items, Page: input.Page, PageSize: input.PageSize,
		Total: total, HasMore: int64(input.Page*input.PageSize) < total,
	}, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (map[string]any, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "用户不存在", nil)
		}
		return nil, err
	}
	base, err := s.userDTO(ctx, user)
	if err != nil {
		return nil, err
	}
	ledgerPage, err := s.ListLedger(ctx, userID, 1, 50)
	if err != nil {
		return nil, err
	}
	codePage, err := s.ListCodes(ctx, CodeQuery{Page: 1, PageSize: 50, RedeemedBy: userID})
	if err != nil {
		return nil, err
	}
	taskPage, err := s.ListTasks(ctx, TaskQuery{Page: 1, PageSize: 50, UserID: userID})
	if err != nil {
		return nil, err
	}
	ticketPage, err := s.ListRetouch(ctx, RetouchQuery{Page: 1, PageSize: 50, UserID: userID})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id": base.ID, "email": base.Email, "status": base.Status,
		"balance": base.Balance, "totalRedeemed": base.TotalRedeemed,
		"totalConsumed": base.TotalConsumed, "taskCount": base.TaskCount,
		"ticketCount": base.TicketCount, "lastLoginAt": base.LastLoginAt,
		"createdAt": base.CreatedAt, "disabledReason": base.DisabledReason,
		"ledger": ledgerPage.Items, "redemptionCodes": codePage.Items,
		"tasks": taskPage.Items, "tickets": ticketPage.Items,
	}, nil
}

func (s *Service) SetUserStatus(
	ctx context.Context,
	userID string,
	status string,
	reason string,
) (*ManagedUserDTO, error) {
	if status != model.UserStatusActive && status != model.UserStatusDisabled {
		return nil, apierror.Invalid("用户状态无效", nil)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apierror.Invalid("状态变更原因不能为空", nil)
	}
	var user model.User
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", userID).Error; err != nil {
			return err
		}
		updates := map[string]any{"status": status, "disabled_reason": "", "disabled_at": nil}
		if status == model.UserStatusDisabled {
			now := time.Now().UTC()
			updates["disabled_reason"] = reason
			updates["disabled_at"] = &now
			if err := tx.Model(&model.UserSession{}).Where("user_id = ? AND revoked_at IS NULL", userID).
				Update("revoked_at", now).Error; err != nil {
				return err
			}
		}
		return tx.Model(&user).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	value, err := s.userDTO(ctx, user)
	return &value, err
}

func (s *Service) ResetPassword(ctx context.Context, userID string) (string, time.Time, error) {
	raw, err := security.RandomToken(12)
	if err != nil {
		return "", time.Time{}, err
	}
	if len(raw) > 16 {
		raw = raw[:16]
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), s.bcryptCost)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	result := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).
		Updates(map[string]any{
			"password_hash": string(hash), "password_reset_required": true,
			"temporary_password_until": &expiresAt,
		})
	if result.Error != nil {
		return "", time.Time{}, result.Error
	}
	if result.RowsAffected != 1 {
		return "", time.Time{}, apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "用户不存在", nil)
	}
	return raw, expiresAt, nil
}

func (s *Service) AdjustCredits(
	ctx context.Context,
	userID string,
	adminID string,
	amount int,
	reason string,
	referenceNo string,
	idempotencyKey string,
) (*ManagedUserDTO, *LedgerDTO, error) {
	reason = strings.TrimSpace(reason)
	if amount == 0 || reason == "" {
		return nil, nil, apierror.Invalid("调次值不能为 0，且必须填写原因", nil)
	}
	var entry *model.CreditLedgerEntry
	replayedLedgerID := ""
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		request := struct {
			Amount      int    `json:"amount"`
			Reason      string `json:"reason"`
			ReferenceNo string `json:"referenceNo"`
		}{Amount: amount, Reason: reason, ReferenceNo: strings.TrimSpace(referenceNo)}
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "manage", PrincipalID: adminID, Method: http.MethodPost,
			Path: "/api/manage/users/" + userID + "/adjust-credits",
			Key:  idempotencyKey, Request: request,
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
			replayedLedgerID = reference.ID
			return nil
		}
		entry, err = s.credits.AddTx(tx, credit.Mutation{
			UserID: userID, Type: credit.LedgerAdjustment, Amount: amount,
			BusinessType: "admin_adjustment", OperatorID: &adminID,
			Reason: reason, ReferenceNo: strings.TrimSpace(referenceNo),
		})
		if err != nil {
			return err
		}
		reference := struct {
			ID string `json:"id"`
		}{ID: entry.ID}
		return idempotency.CompleteTx(
			tx, recordID, http.StatusOK, 0, "credit_ledger_entry", &entry.ID, reference,
		)
	})
	if err != nil {
		return nil, nil, err
	}
	if replayedLedgerID != "" {
		var replayed model.CreditLedgerEntry
		if err := s.db.WithContext(ctx).First(&replayed, "id = ?", replayedLedgerID).Error; err != nil {
			return nil, nil, err
		}
		entry = &replayed
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, nil, err
	}
	userDTO, err := s.userDTO(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	ledger := ledgerDTO(*entry)
	return &userDTO, &ledger, nil
}

func (s *Service) ListLedger(
	ctx context.Context,
	userID string,
	page int,
	pageSize int,
) (PageResult[LedgerDTO], error) {
	page, pageSize = pageValues(page, pageSize)
	query := s.db.WithContext(ctx).Model(&model.CreditLedgerEntry{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PageResult[LedgerDTO]{}, err
	}
	var entries []model.CreditLedgerEntry
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).
		Limit(pageSize).Find(&entries).Error; err != nil {
		return PageResult[LedgerDTO]{}, err
	}
	items := make([]LedgerDTO, 0, len(entries))
	for _, entry := range entries {
		items = append(items, ledgerDTO(entry))
	}
	return PageResult[LedgerDTO]{
		Items: items, Page: page, PageSize: pageSize, Total: total,
		HasMore: int64(page*pageSize) < total,
	}, nil
}

func (s *Service) userDTO(ctx context.Context, user model.User) (ManagedUserDTO, error) {
	var account model.CreditAccount
	if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).First(&account).Error; err != nil {
		return ManagedUserDTO{}, err
	}
	var totalRedeemed int
	if err := s.db.WithContext(ctx).Model(&model.CreditLedgerEntry{}).
		Where("user_id = ? AND type = ?", user.ID, credit.LedgerRedemption).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRedeemed).Error; err != nil {
		return ManagedUserDTO{}, err
	}
	var reservations []model.CreditReservation
	if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).Find(&reservations).Error; err != nil {
		return ManagedUserDTO{}, err
	}
	totalConsumed := 0
	for _, reservation := range reservations {
		totalConsumed += reservation.SettledAmount - reservation.RefundedAmount
	}
	var taskCount, ticketCount int64
	if err := s.db.WithContext(ctx).Model(&model.GenerationTask{}).Where("user_id = ?", user.ID).Count(&taskCount).Error; err != nil {
		return ManagedUserDTO{}, err
	}
	if err := s.db.WithContext(ctx).Model(&model.RetouchTicket{}).Where("user_id = ?", user.ID).Count(&ticketCount).Error; err != nil {
		return ManagedUserDTO{}, err
	}
	var lastLogin *time.Time
	var session model.UserSession
	if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).
		Order("last_seen_at DESC").First(&session).Error; err == nil {
		value := session.LastSeenAt
		lastLogin = &value
	}
	return ManagedUserDTO{
		ID: user.ID, Email: user.Email, Status: user.Status, Balance: account.Balance,
		TotalRedeemed: totalRedeemed, TotalConsumed: totalConsumed,
		TaskCount: taskCount, TicketCount: ticketCount, LastLoginAt: lastLogin,
		CreatedAt: user.CreatedAt, DisabledReason: user.DisabledReason,
	}, nil
}

func ledgerDTO(entry model.CreditLedgerEntry) LedgerDTO {
	description := entry.Reason
	if description == "" {
		description = entry.Type
	}
	return LedgerDTO{
		ID: entry.ID, UserID: entry.UserID, Type: entry.Type, Amount: entry.Amount,
		BalanceBefore: entry.BalanceBefore, BalanceAfter: entry.BalanceAfter,
		Description: description, Reason: entry.Reason, ReferenceNo: entry.ReferenceNo,
		OperatorID: entry.OperatorID, CreatedAt: entry.CreatedAt,
	}
}
