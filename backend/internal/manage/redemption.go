package manage

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/idempotency"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/redemption"
)

type CodeQuery struct {
	Page         int
	PageSize     int
	Search       string
	Status       string
	BatchID      string
	ProductCode  string
	RedeemedBy   string
	ExpiringSoon bool
}

type BatchQuery struct {
	Page        int
	PageSize    int
	Search      string
	ProductCode string
}

func (s *Service) ListCodes(ctx context.Context, input CodeQuery) (PageResult[RedemptionCodeDTO], error) {
	input.Page, input.PageSize = pageValues(input.Page, input.PageSize)
	query := s.db.WithContext(ctx).Model(&model.RedemptionCode{}).
		Joins("LEFT JOIN redemption_batches AS code_batch ON code_batch.id = redemption_codes.batch_id").
		Joins("LEFT JOIN users AS redeeming_user ON redeeming_user.id = redemption_codes.redeemed_by")
	now := time.Now().UTC()
	switch input.Status {
	case redemption.StatusUnused:
		query = query.Where(
			"redemption_codes.redeemed_at IS NULL AND redemption_codes.disabled_at IS NULL "+
				"AND (redemption_codes.expires_at IS NULL OR redemption_codes.expires_at > ?)",
			now,
		)
	case redemption.StatusRedeemed:
		query = query.Where("redemption_codes.redeemed_at IS NOT NULL")
	case redemption.StatusExpired:
		query = query.Where(
			"redemption_codes.redeemed_at IS NULL AND redemption_codes.disabled_at IS NULL "+
				"AND redemption_codes.expires_at <= ?",
			now,
		)
	case redemption.StatusDisabled:
		query = query.Where(
			"redemption_codes.redeemed_at IS NULL AND redemption_codes.disabled_at IS NOT NULL",
		)
	}
	if input.BatchID != "" {
		query = query.Where("redemption_codes.batch_id = ?", input.BatchID)
	}
	if input.ProductCode != "" {
		query = query.Where("redemption_codes.product_code = ?", input.ProductCode)
	}
	if input.RedeemedBy != "" {
		redeemedBy := strings.TrimSpace(input.RedeemedBy)
		query = query.Where(
			"redeeming_user.email ILIKE ? OR CAST(redemption_codes.redeemed_by AS text) = ?",
			redeemedBy,
			redeemedBy,
		)
	}
	if input.ExpiringSoon {
		query = query.Where(
			"redemption_codes.redeemed_at IS NULL AND redemption_codes.disabled_at IS NULL "+
				"AND redemption_codes.expires_at BETWEEN ? AND ?",
			now,
			now.Add(7*24*time.Hour),
		)
	}
	if input.Search != "" {
		search := "%" + strings.TrimSpace(input.Search) + "%"
		query = query.Where(
			"redemption_codes.masked_code ILIKE ? OR code_batch.name ILIKE ? "+
				"OR redeeming_user.email ILIKE ? OR CAST(redemption_codes.id AS text) ILIKE ?",
			search,
			search,
			search,
			search,
		)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PageResult[RedemptionCodeDTO]{}, err
	}
	var codes []model.RedemptionCode
	if err := query.Select("redemption_codes.*").Order("redemption_codes.created_at DESC").
		Offset((input.Page - 1) * input.PageSize).Limit(input.PageSize).
		Find(&codes).Error; err != nil {
		return PageResult[RedemptionCodeDTO]{}, err
	}
	items := make([]RedemptionCodeDTO, 0, len(codes))
	for _, code := range codes {
		value, err := s.codeDTO(ctx, code)
		if err != nil {
			return PageResult[RedemptionCodeDTO]{}, err
		}
		items = append(items, value)
	}
	return PageResult[RedemptionCodeDTO]{
		Items: items, Page: input.Page, PageSize: input.PageSize,
		Total: total, HasMore: int64(input.Page*input.PageSize) < total,
	}, nil
}

func (s *Service) GetCode(ctx context.Context, codeID string) (map[string]any, error) {
	var code model.RedemptionCode
	if err := s.db.WithContext(ctx).First(&code, "id = ?", codeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierror.Invalid("兑换码不存在", nil)
		}
		return nil, err
	}
	dto, err := s.codeDTO(ctx, code)
	if err != nil {
		return nil, err
	}
	history := []map[string]any{{
		"action": "generated", "operator": dto.DisabledBy, "createdAt": code.CreatedAt,
	}}
	if code.RedeemedAt != nil {
		history = append(history, map[string]any{
			"action": "redeemed", "operator": dto.RedeemedByEmail, "createdAt": code.RedeemedAt,
		})
	}
	if code.DisabledAt != nil {
		history = append(history, map[string]any{
			"action": "disabled", "operator": dto.DisabledBy,
			"reason": code.DisabledReason, "createdAt": code.DisabledAt,
		})
	}
	encoded := map[string]any{
		"id": dto.ID, "maskedCode": dto.MaskedCode, "batchId": dto.BatchID,
		"batchName": dto.BatchName, "productCode": dto.ProductCode, "credits": dto.Credits,
		"status": dto.Status, "expiresAt": dto.ExpiresAt, "redeemedBy": dto.RedeemedBy,
		"redeemedByEmail": dto.RedeemedByEmail, "redeemedAt": dto.RedeemedAt,
		"disabledBy": dto.DisabledBy, "disabledAt": dto.DisabledAt,
		"disabledReason": dto.DisabledReason, "createdAt": dto.CreatedAt,
		"expiringSoon": dto.ExpiringSoon, "operationHistory": history,
	}
	return encoded, nil
}

func (s *Service) ListBatches(ctx context.Context, input BatchQuery) (PageResult[RedemptionBatchDTO], error) {
	input.Page, input.PageSize = pageValues(input.Page, input.PageSize)
	query := s.db.WithContext(ctx).Model(&model.RedemptionBatch{})
	if input.Search != "" {
		query = query.Where("name ILIKE ?", "%"+strings.TrimSpace(input.Search)+"%")
	}
	if input.ProductCode != "" {
		query = query.Where("product_code = ?", input.ProductCode)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PageResult[RedemptionBatchDTO]{}, err
	}
	var batches []model.RedemptionBatch
	if err := query.Order("created_at DESC").
		Offset((input.Page - 1) * input.PageSize).Limit(input.PageSize).
		Find(&batches).Error; err != nil {
		return PageResult[RedemptionBatchDTO]{}, err
	}
	items := make([]RedemptionBatchDTO, 0, len(batches))
	for _, batch := range batches {
		value, err := s.batchDTO(ctx, batch)
		if err != nil {
			return PageResult[RedemptionBatchDTO]{}, err
		}
		items = append(items, value)
	}
	return PageResult[RedemptionBatchDTO]{
		Items: items, Page: input.Page, PageSize: input.PageSize,
		Total: total, HasMore: int64(input.Page*input.PageSize) < total,
	}, nil
}

func (s *Service) GetBatch(ctx context.Context, batchID string) (*RedemptionBatchDTO, error) {
	var batch model.RedemptionBatch
	if err := s.db.WithContext(ctx).First(&batch, "id = ?", batchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "兑换码批次不存在", nil)
		}
		return nil, err
	}
	value, err := s.batchDTO(ctx, batch)
	return &value, err
}

func (s *Service) UpdateBatchName(
	ctx context.Context,
	adminID string,
	batchID string,
	name string,
	idempotencyKey string,
) (*RedemptionBatchDTO, string, error) {
	var err error
	name, err = redemption.NormalizeBatchName(name)
	if err != nil {
		return nil, "", err
	}

	previousName := ""
	responseName := name
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		request := struct {
			Name string `json:"name"`
		}{Name: name}
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "manage",
			PrincipalID:    adminID,
			Method:         http.MethodPatch,
			Path:           "/api/manage/redemption-batches/" + batchID,
			Key:            idempotencyKey,
			Request:        request,
		})
		if err != nil {
			return err
		}
		if replay != nil {
			var reference struct {
				ID           string `json:"id"`
				PreviousName string `json:"previousName"`
				Name         string `json:"name"`
			}
			if err := json.Unmarshal(replay.Data, &reference); err != nil {
				return err
			}
			previousName = reference.PreviousName
			responseName = reference.Name
			return nil
		}

		var batch model.RedemptionBatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&batch, "id = ?", batchID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apierror.New(http.StatusNotFound, apierror.CodeInvalidInput, "兑换码批次不存在", nil)
			}
			return err
		}
		previousName = batch.Name
		if previousName != name {
			if err := tx.Model(&batch).Update("name", name).Error; err != nil {
				return err
			}
		}
		reference := struct {
			ID           string `json:"id"`
			PreviousName string `json:"previousName"`
			Name         string `json:"name"`
		}{ID: batch.ID, PreviousName: previousName, Name: name}
		return idempotency.CompleteTx(
			tx, recordID, http.StatusOK, 0, "redemption_batch", &batch.ID, reference,
		)
	})
	if err != nil {
		return nil, previousName, err
	}
	value, err := s.GetBatch(ctx, batchID)
	if value != nil {
		value.Name = responseName
	}
	return value, previousName, err
}

func (s *Service) RevealBatch(ctx context.Context, batchID string) ([]redemption.GeneratedCode, error) {
	var codes []model.RedemptionCode
	if err := s.db.WithContext(ctx).Where("batch_id = ?", batchID).Order("created_at ASC").Find(&codes).Error; err != nil {
		return nil, err
	}
	result := make([]redemption.GeneratedCode, 0)
	for _, code := range codes {
		if redemption.Status(code, time.Now().UTC()) != redemption.StatusUnused {
			continue
		}
		full, err := s.redemptions.Reveal(code.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, redemption.GeneratedCode{ID: code.ID, FullCode: full, MaskedCode: code.MaskedCode})
	}
	return result, nil
}

func (s *Service) ExportBatch(ctx context.Context, batchID string) (string, string, error) {
	batch, err := s.GetBatch(ctx, batchID)
	if err != nil {
		return "", "", err
	}
	codes, err := s.RevealBatch(ctx, batchID)
	if err != nil {
		return "", "", err
	}
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	_ = writer.Write([]string{"code", "credits", "product_code", "expires_at"})
	expiresAt := ""
	if batch.ExpiresAt != nil {
		expiresAt = batch.ExpiresAt.UTC().Format(time.RFC3339)
	}
	for _, code := range codes {
		_ = writer.Write([]string{code.FullCode, fmt.Sprint(batch.CreditsPerCode), batch.ProductCode, expiresAt})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", "", err
	}
	return "redemption-" + batch.ID + ".csv", builder.String(), nil
}

func (s *Service) batchDTO(ctx context.Context, batch model.RedemptionBatch) (RedemptionBatchDTO, error) {
	var codes []model.RedemptionCode
	if err := s.db.WithContext(ctx).Where("batch_id = ?", batch.ID).Find(&codes).Error; err != nil {
		return RedemptionBatchDTO{}, err
	}
	counts := map[string]int{
		redemption.StatusUnused: 0, redemption.StatusRedeemed: 0,
		redemption.StatusExpired: 0, redemption.StatusDisabled: 0,
	}
	for _, code := range codes {
		counts[redemption.Status(code, time.Now().UTC())]++
	}
	usageRate := 0.0
	if batch.Quantity > 0 {
		usageRate = float64(counts[redemption.StatusRedeemed]) / float64(batch.Quantity)
	}
	return RedemptionBatchDTO{
		ID: batch.ID, Name: batch.Name, ProductCode: batch.ProductCode,
		Quantity: batch.Quantity, CreditsPerCode: batch.CreditsPerCode,
		ExpiresAt: batch.ExpiresAt, NeverExpires: batch.ExpiresAt == nil,
		Note: batch.Notes, CreatedBy: batch.CreatedBy, CreatedAt: batch.CreatedAt,
		Counts: counts, UsageRate: usageRate,
	}, nil
}

func (s *Service) codeDTO(ctx context.Context, code model.RedemptionCode) (RedemptionCodeDTO, error) {
	var batch model.RedemptionBatch
	if err := s.db.WithContext(ctx).First(&batch, "id = ?", code.BatchID).Error; err != nil {
		return RedemptionCodeDTO{}, err
	}
	value := RedemptionCodeDTO{
		ID: code.ID, MaskedCode: code.MaskedCode, BatchID: code.BatchID,
		BatchName: batch.Name, ProductCode: code.ProductCode, Credits: code.Credits,
		Status: redemption.Status(code, time.Now().UTC()), ExpiresAt: code.ExpiresAt,
		RedeemedAt: code.RedeemedAt, DisabledAt: code.DisabledAt,
		DisabledReason: code.DisabledReason, CreatedAt: code.CreatedAt,
		ExpiringSoon: code.ExpiresAt != nil && code.ExpiresAt.After(time.Now().UTC()) &&
			code.ExpiresAt.Before(time.Now().UTC().Add(7*24*time.Hour)),
	}
	if code.RedeemedBy != nil {
		value.RedeemedBy = *code.RedeemedBy
		var user model.User
		if s.db.WithContext(ctx).First(&user, "id = ?", *code.RedeemedBy).Error == nil {
			value.RedeemedByEmail = user.Email
		}
	}
	if code.DisabledBy != nil {
		value.DisabledBy = *code.DisabledBy
		var admin model.AdminAccount
		if s.db.WithContext(ctx).First(&admin, "id = ?", *code.DisabledBy).Error == nil {
			value.DisabledBy = admin.Email
		}
	}
	return value, nil
}
