package notice

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/idempotency"
	"yingyan.local/backend/internal/model"
)

const CurrentVersion = "ai-processing-v2"

type DTO struct {
	Version            string     `json:"version"`
	Title              string     `json:"title"`
	ProviderDisclosure string     `json:"providerDisclosure"`
	SecuritySummary    string     `json:"securitySummary"`
	Purpose            string     `json:"purpose"`
	ProcessingScope    []string   `json:"processingScope"`
	RetentionDays      int        `json:"retentionDays"`
	StopUseDescription string     `json:"stopUseDescription"`
	Acknowledged       bool       `json:"acknowledged"`
	AcknowledgedAt     *time.Time `json:"acknowledgedAt,omitempty"`
}

type Service struct{ db *gorm.DB }

func New(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Get(ctx context.Context, userID string) (*DTO, error) {
	var record model.UserAIProcessingNotice
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&record).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return dto(record), nil
}

func (s *Service) IsAcknowledged(ctx context.Context, userID string) (bool, error) {
	var record model.UserAIProcessingNotice
	err := s.db.WithContext(ctx).Where("user_id = ? AND notice_version = ?", userID, CurrentVersion).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (s *Service) Ack(ctx context.Context, userID string, version string, key string) (*DTO, error) {
	if version == "" {
		version = CurrentVersion
	}
	if version != CurrentVersion {
		return nil, apierror.New(http.StatusConflict, apierror.CodeInvalidInput, "告知版本已更新，请重新加载", map[string]string{"version": CurrentVersion})
	}
	var result DTO
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recordID, replay, err := idempotency.AcquireTx(tx, idempotency.Scope{
			PrincipalRealm: "user", PrincipalID: userID, Method: http.MethodPost,
			Path: "/api/notices/ai-processing/ack", Key: key,
			Request: struct {
				Version string `json:"version"`
			}{version},
		})
		if err != nil {
			return err
		}
		if replay != nil {
			return json.Unmarshal(replay.Data, &result)
		}
		now := time.Now().UTC()
		var record model.UserAIProcessingNotice
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record = model.UserAIProcessingNotice{BaseModel: model.BaseModel{ID: ""}, UserID: userID}
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if record.ID == "" {
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&record).Updates(map[string]any{
			"notice_version":  CurrentVersion,
			"acknowledged_at": now,
		}).Error; err != nil {
			return err
		}
		result = *dto(record)
		result.Version = CurrentVersion
		result.Acknowledged = true
		result.AcknowledgedAt = &now
		return idempotency.CompleteTx(tx, recordID, http.StatusOK, 0, "ai_processing_notice", &record.ID, result)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func dto(record model.UserAIProcessingNotice) *DTO {
	acknowledged := record.NoticeVersion == CurrentVersion && !record.AcknowledgedAt.IsZero()
	var acknowledgedAt *time.Time
	if acknowledged {
		value := record.AcknowledgedAt.UTC()
		acknowledgedAt = &value
	}
	return &DTO{
		Version:            CurrentVersion,
		Title:              "关于第三方 AI 处理的告知",
		ProviderDisclosure: "当您主动使用提示词优化或图片生成功能时，映研只会把完成当前任务所需的图片和提示词发送给平台配置的第三方 AI 服务商；不会公开展示您的内容，也不会提供给其他用户。",
		SecuritySummary:    "平台采用权限隔离、私有存储和短期签名地址保护您的素材。服务商只收到当前任务所需的内容；您的密码、兑换码、次数余额等不会发送给服务商。平台 API Key 由服务端保管，不会向其他用户展示或写入普通日志。",
		Purpose:            "仅用于完成您主动点击的提示词优化、文生图和图生图任务，不用于广告或向其他用户展示。",
		ProcessingScope:    []string{"当前任务所需的原图和参考图（参考图仅用于提示词分析，不会作为图生图输入）", "当前任务的需求、已确认提示词和生成参数", "不会发送您的账号密码、兑换码、次数余额或其他用户信息"},
		RetentionDays:      90,
		StopUseDescription: "素材默认在任务完成或结束后保留 90 天，用于查看结果和处理关联服务。您可以随时停止后续使用；已经发送给第三方服务商的请求无法撤回。",
		Acknowledged:       acknowledged,
		AcknowledgedAt:     acknowledgedAt,
	}
}
