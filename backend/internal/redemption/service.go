package redemption

import (
	"context"
	"crypto/rand"
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
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/idempotency"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/platform/security"
)

const (
	StatusUnused       = "unused"
	StatusRedeemed     = "redeemed"
	StatusExpired      = "expired"
	StatusDisabled     = "disabled"
	BatchNameMaxLength = 60
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func NormalizeBatchName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apierror.Invalid("批次名称不能为空", map[string]any{"field": "name"})
	}
	if len([]rune(name)) > BatchNameMaxLength {
		return "", apierror.Invalid("批次名称不能超过 60 个字符", map[string]any{"field": "name"})
	}
	return name, nil
}

type Service struct {
	db                *gorm.DB
	credits           *credit.Service
	encryptionKey     string
	redemptionPepper  string
	clientProductCode string
	clientProductName string
}

type Entitlement struct {
	Balance   int    `json:"balance"`
	CanCreate bool   `json:"canCreate"`
	Status    string `json:"status"`
}

type ClaimResult struct {
	Added       int         `json:"added"`
	Entitlement Entitlement `json:"entitlement"`
}

type PreviewResult struct {
	Valid       bool       `json:"valid"`
	Credits     int        `json:"credits"`
	ProductName string     `json:"productName"`
	MaskedCode  string     `json:"maskedCode"`
	ExpiresAt   *time.Time `json:"expiresAt"`
}

type CreateBatchInput struct {
	Name           string
	Quantity       int
	CreditsPerCode int
	ProductCode    string
	ExpiresAt      *time.Time
	Note           string
	CreatedBy      string
}

type GeneratedCode struct {
	ID         string `json:"id"`
	FullCode   string `json:"fullCode"`
	MaskedCode string `json:"maskedCode"`
}

type CreateBatchResult struct {
	Batch model.RedemptionBatch `json:"batch"`
	Codes []GeneratedCode       `json:"codes"`
}

func New(
	db *gorm.DB,
	credits *credit.Service,
	encryptionKey string,
	redemptionPepper string,
	clientProductCode string,
	clientProductName string,
) *Service {
	return &Service{
		db:                db,
		credits:           credits,
		encryptionKey:     encryptionKey,
		redemptionPepper:  redemptionPepper,
		clientProductCode: strings.TrimSpace(clientProductCode),
		clientProductName: strings.TrimSpace(clientProductName),
	}
}

func (s *Service) Preview(ctx context.Context, rawCode string) (*PreviewResult, error) {
	normalized := security.NormalizeRedemptionCode(rawCode)
	if normalized == "" {
		return nil, apierror.Invalid("请输入兑换码", nil)
	}

	var code model.RedemptionCode
	err := s.db.WithContext(ctx).
		Where("code_digest = ?", security.HMACDigest(normalized, s.redemptionPepper)).
		First(&code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, redemptionInvalid()
	}
	if err != nil {
		return nil, err
	}
	if err := s.validateCode(code, "", false); err != nil {
		return nil, err
	}
	return &PreviewResult{
		Valid:       true,
		Credits:     code.Credits,
		ProductName: s.clientProductName,
		MaskedCode:  code.MaskedCode,
		ExpiresAt:   code.ExpiresAt,
	}, nil
}

func Status(code model.RedemptionCode, now time.Time) string {
	switch {
	case code.RedeemedAt != nil:
		return StatusRedeemed
	case code.DisabledAt != nil:
		return StatusDisabled
	case code.ExpiresAt != nil && !code.ExpiresAt.After(now):
		return StatusExpired
	default:
		return StatusUnused
	}
}

func (s *Service) Claim(
	userID string,
	rawCode string,
	idempotencyKey string,
) (*ClaimResult, error) {
	normalized := security.NormalizeRedemptionCode(rawCode)
	if normalized == "" {
		return nil, apierror.Invalid("请输入兑换码", nil)
	}
	request := struct {
		Code string `json:"code"`
	}{Code: normalized}

	var result ClaimResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "user",
			PrincipalID:    userID,
			Method:         http.MethodPost,
			Path:           "/api/redemptions/claim",
			Key:            idempotencyKey,
			Request:        request,
		})
		if err != nil {
			return err
		}
		if replay != nil {
			if replay.Code != 0 {
				return apierror.New(replay.HTTPStatus, replay.Code, "兑换失败", replay.Data)
			}
			return json.Unmarshal(replay.Data, &result)
		}

		digest := security.HMACDigest(normalized, s.redemptionPepper)
		var code model.RedemptionCode
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code_digest = ?", digest).
			First(&code).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return redemptionInvalid()
		}
		if err != nil {
			return err
		}

		if err := s.validateCode(code, userID, true); err != nil {
			return err
		}

		now := time.Now().UTC()
		update := tx.Model(&code).
			Where("redeemed_at IS NULL AND disabled_at IS NULL").
			Updates(map[string]any{
				"redeemed_at": &now,
				"redeemed_by": userID,
				"version":     gorm.Expr("version + 1"),
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return apierror.New(
				http.StatusConflict,
				apierror.CodeRedemptionUsed,
				"兑换码已使用",
				nil,
			)
		}

		codeID := code.ID
		ledger, err := s.credits.AddTx(tx, credit.Mutation{
			UserID:       userID,
			Type:         credit.LedgerRedemption,
			Amount:       code.Credits,
			BusinessType: "redemption",
			BusinessID:   &codeID,
			Reason:       "兑换码充值",
			ReferenceNo:  code.MaskedCode,
		})
		if err != nil {
			return err
		}

		claim := model.RedemptionClaim{
			CodeID:         code.ID,
			UserID:         userID,
			CreditsGranted: code.Credits,
			LedgerEntryID:  ledger.ID,
			IdempotencyKey: idempotencyKey,
		}
		if err := tx.Create(&claim).Error; err != nil {
			return err
		}

		result = ClaimResult{
			Added:       code.Credits,
			Entitlement: entitlementFromBalance(ledger.BalanceAfter),
		}
		return idempotency.CompleteTx(
			tx,
			recordID,
			http.StatusOK,
			0,
			"redemption_claim",
			&claim.ID,
			result,
		)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) validateCode(code model.RedemptionCode, userID string, revealOwnership bool) error {
	if code.ProductCode != s.clientProductCode {
		return apierror.New(
			http.StatusConflict,
			apierror.CodeProductMismatch,
			"兑换码不适用于当前商品",
			nil,
		)
	}
	switch Status(code, time.Now().UTC()) {
	case StatusRedeemed:
		details := map[string]any(nil)
		if revealOwnership && userID != "" && code.RedeemedBy != nil && *code.RedeemedBy == userID {
			details = map[string]any{"claimedByCurrentUser": true}
		}
		return apierror.New(http.StatusConflict, apierror.CodeRedemptionUsed, "兑换码已使用", details)
	case StatusExpired:
		return apierror.New(http.StatusGone, apierror.CodeRedemptionExpired, "兑换码已过期", nil)
	case StatusDisabled:
		return redemptionInvalid()
	default:
		return nil
	}
}

func redemptionInvalid() error {
	return apierror.New(http.StatusNotFound, apierror.CodeRedemptionInvalid, "兑换码无效", nil)
}

func (s *Service) CreateBatch(
	ctx context.Context,
	input CreateBatchInput,
	idempotencyKey string,
) (*CreateBatchResult, error) {
	var err error
	input.Name, err = NormalizeBatchName(input.Name)
	if err != nil {
		return nil, err
	}
	input.ProductCode = strings.TrimSpace(input.ProductCode)
	input.Note = strings.TrimSpace(input.Note)
	if input.ProductCode == "" {
		return nil, apierror.Invalid("商品标识不能为空", nil)
	}
	if input.Quantity < 1 || input.Quantity > 500 {
		return nil, apierror.Invalid("单批兑换码数量必须为 1 到 500", nil)
	}
	if input.CreditsPerCode <= 0 {
		return nil, apierror.Invalid("每码次数必须大于 0", nil)
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now().UTC()) {
		return nil, apierror.Invalid("有效期必须晚于当前时间", nil)
	}

	var output CreateBatchResult
	replayedBatchID := ""
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "manage",
			PrincipalID:    input.CreatedBy,
			Method:         http.MethodPost,
			Path:           "/api/manage/redemption-batches",
			Key:            idempotencyKey,
			Request: struct {
				Name           string     `json:"name"`
				Quantity       int        `json:"quantity"`
				CreditsPerCode int        `json:"creditsPerCode"`
				ProductCode    string     `json:"productCode"`
				ExpiresAt      *time.Time `json:"expiresAt"`
				Note           string     `json:"note"`
			}{
				Name: input.Name, Quantity: input.Quantity,
				CreditsPerCode: input.CreditsPerCode, ProductCode: input.ProductCode,
				ExpiresAt: input.ExpiresAt, Note: input.Note,
			},
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
			replayedBatchID = reference.ID
			return nil
		}
		batch := model.RedemptionBatch{
			Name:           input.Name,
			Quantity:       input.Quantity,
			CreditsPerCode: input.CreditsPerCode,
			ProductCode:    input.ProductCode,
			ExpiresAt:      input.ExpiresAt,
			Notes:          input.Note,
			CreatedBy:      input.CreatedBy,
		}
		if err := tx.Create(&batch).Error; err != nil {
			return err
		}

		output.Batch = batch
		output.Codes = make([]GeneratedCode, 0, input.Quantity)
		for index := 0; index < input.Quantity; index++ {
			fullCode, err := s.newUniqueCode(tx)
			if err != nil {
				return err
			}
			ciphertext, err := security.Encrypt([]byte(fullCode), s.encryptionKey)
			if err != nil {
				return err
			}
			code := model.RedemptionCode{
				BatchID:        batch.ID,
				CodeDigest:     security.HMACDigest(fullCode, s.redemptionPepper),
				CodeCiphertext: ciphertext,
				MaskedCode:     maskCode(fullCode),
				Credits:        input.CreditsPerCode,
				ProductCode:    input.ProductCode,
				ExpiresAt:      input.ExpiresAt,
				Version:        1,
			}
			if err := tx.Create(&code).Error; err != nil {
				return err
			}
			output.Codes = append(output.Codes, GeneratedCode{
				ID:         code.ID,
				FullCode:   fullCode,
				MaskedCode: code.MaskedCode,
			})
		}
		reference := struct {
			ID string `json:"id"`
		}{ID: batch.ID}
		return idempotency.CompleteTx(
			tx, recordID, http.StatusCreated, 0, "redemption_batch", &batch.ID, reference,
		)
	})
	if err != nil {
		return nil, err
	}
	if replayedBatchID != "" {
		var batch model.RedemptionBatch
		if err := s.db.WithContext(ctx).First(&batch, "id = ?", replayedBatchID).Error; err != nil {
			return nil, err
		}
		var codes []model.RedemptionCode
		if err := s.db.WithContext(ctx).Where("batch_id = ?", batch.ID).
			Order("created_at ASC").Find(&codes).Error; err != nil {
			return nil, err
		}
		output.Batch = batch
		output.Codes = make([]GeneratedCode, 0, len(codes))
		for _, code := range codes {
			plaintext, err := security.Decrypt(code.CodeCiphertext, s.encryptionKey)
			if err != nil {
				return nil, err
			}
			output.Codes = append(output.Codes, GeneratedCode{
				ID: code.ID, FullCode: string(plaintext), MaskedCode: code.MaskedCode,
			})
		}
	}
	return &output, nil
}

func (s *Service) Reveal(codeID string) (string, error) {
	var code model.RedemptionCode
	if err := s.db.First(&code, "id = ?", codeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apierror.New(http.StatusNotFound, apierror.CodeRedemptionInvalid, "兑换码不存在", nil)
		}
		return "", err
	}
	if Status(code, time.Now().UTC()) != StatusUnused {
		return "", apierror.New(
			http.StatusConflict,
			apierror.CodeInvalidInput,
			"仅未使用兑换码可以查看完整值",
			nil,
		)
	}
	plaintext, err := security.Decrypt(code.CodeCiphertext, s.encryptionKey)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *Service) Disable(
	ctx context.Context,
	codeIDs []string,
	batchID string,
	adminID string,
	reason string,
	idempotencyKey string,
) (affected int64, skipped int64, err error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return 0, 0, apierror.Invalid("失效原因不能为空", nil)
	}
	if len(codeIDs) == 0 && batchID == "" {
		return 0, 0, apierror.Invalid("请选择兑换码或批次", nil)
	}

	request := struct {
		CodeIDs []string `json:"codeIds"`
		BatchID string   `json:"batchId"`
		Reason  string   `json:"reason"`
	}{CodeIDs: codeIDs, BatchID: batchID, Reason: reason}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "manage", PrincipalID: adminID, Method: http.MethodPost,
			Path: "/api/manage/redemption-codes/disable", Key: idempotencyKey, Request: request,
		})
		if err != nil {
			return err
		}
		if replay != nil {
			var result struct {
				Affected int64 `json:"affected"`
				Skipped  int64 `json:"skipped"`
			}
			if err := json.Unmarshal(replay.Data, &result); err != nil {
				return err
			}
			affected, skipped = result.Affected, result.Skipped
			return nil
		}
		query := tx.Model(&model.RedemptionCode{})
		if batchID != "" {
			query = query.Where("batch_id = ?", batchID)
		} else {
			query = query.Where("id IN ?", codeIDs)
		}
		var total int64
		if err := query.Count(&total).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		result := query.
			Where("redeemed_at IS NULL AND disabled_at IS NULL").
			Where("expires_at IS NULL OR expires_at > ?", now).
			Updates(map[string]any{
				"disabled_at":     &now,
				"disabled_by":     adminID,
				"disabled_reason": reason,
				"version":         gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		affected = result.RowsAffected
		skipped = total - affected
		return idempotency.CompleteTx(
			tx, recordID, http.StatusOK, 0, "redemption_code", nil,
			map[string]int64{"affected": affected, "skipped": skipped},
		)
	})
	return affected, skipped, err
}

func (s *Service) Extend(
	ctx context.Context,
	codeIDs []string,
	batchID string,
	expiresAt *time.Time,
	adminID string,
	reason string,
	idempotencyKey string,
) (affected int64, skipped int64, err error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return 0, 0, apierror.Invalid("延期原因不能为空", nil)
	}
	if expiresAt != nil && !expiresAt.After(time.Now().UTC()) {
		return 0, 0, apierror.Invalid("新的有效期必须晚于当前时间", nil)
	}
	if len(codeIDs) == 0 && batchID == "" {
		return 0, 0, apierror.Invalid("请选择兑换码或批次", nil)
	}

	request := struct {
		CodeIDs   []string   `json:"codeIds"`
		BatchID   string     `json:"batchId"`
		ExpiresAt *time.Time `json:"expiresAt"`
		Reason    string     `json:"reason"`
	}{CodeIDs: codeIDs, BatchID: batchID, ExpiresAt: expiresAt, Reason: reason}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "manage", PrincipalID: adminID, Method: http.MethodPost,
			Path: "/api/manage/redemption-codes/extend", Key: idempotencyKey, Request: request,
		})
		if err != nil {
			return err
		}
		if replay != nil {
			var result struct {
				Affected int64 `json:"affected"`
				Skipped  int64 `json:"skipped"`
			}
			if err := json.Unmarshal(replay.Data, &result); err != nil {
				return err
			}
			affected, skipped = result.Affected, result.Skipped
			return nil
		}
		query := tx.Model(&model.RedemptionCode{})
		if batchID != "" {
			query = query.Where("batch_id = ?", batchID)
		} else {
			query = query.Where("id IN ?", codeIDs)
		}
		var total int64
		if err := query.Count(&total).Error; err != nil {
			return err
		}
		result := query.
			Where("redeemed_at IS NULL AND disabled_at IS NULL").
			Update("expires_at", expiresAt)
		if result.Error != nil {
			return result.Error
		}
		affected = result.RowsAffected
		skipped = total - affected
		return idempotency.CompleteTx(
			tx, recordID, http.StatusOK, 0, "redemption_code", nil,
			map[string]int64{"affected": affected, "skipped": skipped},
		)
	})
	return affected, skipped, err
}

func (s *Service) newUniqueCode(tx *gorm.DB) (string, error) {
	for attempt := 0; attempt < 10; attempt++ {
		parts := make([]string, 3)
		for part := range parts {
			value, err := randomCharacters(4)
			if err != nil {
				return "", err
			}
			parts[part] = value
		}
		fullCode := "YY-" + strings.Join(parts, "-")
		digest := security.HMACDigest(fullCode, s.redemptionPepper)
		var count int64
		if err := tx.Model(&model.RedemptionCode{}).
			Where("code_digest = ?", digest).
			Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return fullCode, nil
		}
	}
	return "", fmt.Errorf("generate unique redemption code: exhausted retries")
}

func randomCharacters(length int) (string, error) {
	buffer := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for index, value := range random {
		buffer[index] = codeAlphabet[int(value)%len(codeAlphabet)]
	}
	return string(buffer), nil
}

func maskCode(code string) string {
	parts := strings.Split(code, "-")
	if len(parts) != 4 {
		return security.MaskSecret(code)
	}
	return strings.Join([]string{parts[0], parts[1], "****", parts[3]}, "-")
}

func entitlementFromBalance(balance int) Entitlement {
	status := "empty"
	if balance > 0 {
		status = "active"
	}
	return Entitlement{
		Balance:   balance,
		CanCreate: balance > 0,
		Status:    status,
	}
}

func NewBusinessID() string {
	return uuid.NewString()
}
