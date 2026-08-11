package audit

import (
	"context"
	"encoding/json"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/platform/security"
)

type Service struct {
	db     *gorm.DB
	pepper string
}

type Entry struct {
	AdminID      *string
	AdminEmail   string
	AdminRole    string
	Action       string
	ResourceType string
	ResourceID   *string
	Before       any
	After        any
	Reason       string
	Result       string
	RequestID    string
	IPAddress    string
	UserAgent    string
}

func New(db *gorm.DB, pepper string) *Service {
	return &Service{db: db, pepper: pepper}
}

func (s *Service) Record(ctx context.Context, entry Entry) error {
	before, _ := json.Marshal(Sanitize(entry.Before))
	after, _ := json.Marshal(Sanitize(entry.After))
	value := model.AuditLog{
		AdminID:       entry.AdminID,
		AdminEmail:    entry.AdminEmail,
		AdminRole:     entry.AdminRole,
		Action:        entry.Action,
		ResourceType:  entry.ResourceType,
		ResourceID:    entry.ResourceID,
		BeforeSummary: datatypes.JSON(before),
		AfterSummary:  datatypes.JSON(after),
		Reason:        strings.TrimSpace(entry.Reason),
		Result:        entry.Result,
		RequestID:     entry.RequestID,
		IPAddressHash: security.HashMetadata(entry.IPAddress, s.pepper),
		UserAgentHash: security.HashMetadata(entry.UserAgent, s.pepper),
	}
	return s.db.WithContext(ctx).Create(&value).Error
}

func Sanitize(value any) any {
	encoded, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	var normalized any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return map[string]any{}
	}
	return sanitizeValue(normalized)
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			lower := strings.ToLower(key)
			if isSensitiveKey(lower) {
				result[key] = "[REDACTED]"
				continue
			}
			result[key] = sanitizeValue(item)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = sanitizeValue(item)
		}
		return result
	case string:
		if strings.HasPrefix(typed, "http") && strings.Contains(typed, "X-Amz-Signature=") {
			return "[SIGNED_URL_REDACTED]"
		}
		return typed
	default:
		return typed
	}
}

func isSensitiveKey(key string) bool {
	for _, fragment := range []string{
		"password", "apikey", "api_key", "ciphertext", "token", "fullcode",
		"full_code", "objectkey", "object_key", "signedurl", "signed_url",
	} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}
