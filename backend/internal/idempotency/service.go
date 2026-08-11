package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/model"
)

const pendingResponseCode = -1

type Scope struct {
	PrincipalRealm string
	PrincipalID    string
	Method         string
	Path           string
	Key            string
	Request        any
}

type Replay struct {
	HTTPStatus int
	Code       int
	Data       json.RawMessage
}

func RequestDigest(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func Lookup(db *gorm.DB, scope Scope) (*Replay, error) {
	if scope.Key == "" {
		return nil, apierror.Invalid("缺少 Idempotency-Key", nil)
	}
	digest, err := RequestDigest(scope.Request)
	if err != nil {
		return nil, apierror.Internal(err)
	}

	var existing model.IdempotencyRecord
	err = db.Where(
		"principal_realm = ? AND principal_id = ? AND method = ? AND path = ? AND key = ?",
		scope.PrincipalRealm,
		scope.PrincipalID,
		scope.Method,
		scope.Path,
		scope.Key,
	).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if existing.RequestDigest != digest {
		return nil, apierror.New(
			http.StatusConflict,
			apierror.CodeIdempotencyConflict,
			"同一个幂等键不能用于不同请求",
			nil,
		)
	}
	if existing.ResponseCode == pendingResponseCode {
		return nil, apierror.New(
			http.StatusConflict,
			apierror.CodeIdempotencyConflict,
			"请求正在处理中，请稍后重试",
			nil,
		)
	}
	return &Replay{
		HTTPStatus: existing.HTTPStatus,
		Code:       existing.ResponseCode,
		Data:       json.RawMessage(existing.ResponseData),
	}, nil
}

// AcquireTx creates the idempotency placeholder in the caller's transaction.
// PostgreSQL's unique index makes a concurrent identical request wait for the
// first transaction, then return its stored response.
func AcquireTx(tx *gorm.DB, scope Scope) (string, *Replay, error) {
	if scope.Key == "" {
		return "", nil, apierror.Invalid("缺少 Idempotency-Key", nil)
	}

	digest, err := RequestDigest(scope.Request)
	if err != nil {
		return "", nil, apierror.Internal(err)
	}

	record := model.IdempotencyRecord{
		PrincipalRealm: scope.PrincipalRealm,
		PrincipalID:    scope.PrincipalID,
		Method:         scope.Method,
		Path:           scope.Path,
		Key:            scope.Key,
		RequestDigest:  digest,
		HTTPStatus:     0,
		ResponseCode:   pendingResponseCode,
		ResponseData:   datatypes.JSON([]byte("null")),
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
	if result.Error != nil {
		return "", nil, result.Error
	}
	if result.RowsAffected == 1 {
		return record.ID, nil, nil
	}

	var existing model.IdempotencyRecord
	err = tx.Where(
		"principal_realm = ? AND principal_id = ? AND method = ? AND path = ? AND key = ?",
		scope.PrincipalRealm,
		scope.PrincipalID,
		scope.Method,
		scope.Path,
		scope.Key,
	).First(&existing).Error
	if err != nil {
		return "", nil, err
	}
	if existing.RequestDigest != digest {
		return "", nil, apierror.New(
			http.StatusConflict,
			apierror.CodeIdempotencyConflict,
			"同一个幂等键不能用于不同请求",
			nil,
		)
	}
	if existing.ResponseCode == pendingResponseCode {
		return "", nil, apierror.New(
			http.StatusConflict,
			apierror.CodeIdempotencyConflict,
			"请求正在处理中，请稍后重试",
			nil,
		)
	}
	return existing.ID, &Replay{
		HTTPStatus: existing.HTTPStatus,
		Code:       existing.ResponseCode,
		Data:       json.RawMessage(existing.ResponseData),
	}, nil
}

func CompleteTx(
	tx *gorm.DB,
	recordID string,
	httpStatus int,
	code int,
	resourceType string,
	resourceID *string,
	data any,
) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	result := tx.Model(&model.IdempotencyRecord{}).
		Where("id = ? AND response_code = ?", recordID, pendingResponseCode).
		Updates(map[string]any{
			"http_status":   httpStatus,
			"response_code": code,
			"response_data": datatypes.JSON(payload),
			"resource_type": resourceType,
			"resource_id":   resourceID,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("idempotency record was already completed")
	}
	return nil
}
