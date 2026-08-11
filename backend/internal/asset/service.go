package asset

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/storage"
)

const (
	KindSource           = "source"
	KindReference        = "reference"
	KindRetouchReference = "retouch-reference"
	KindAIResult         = "ai-result"
	KindRetouchResult    = "retouch-result"

	maxUserUploadSize  = int64(15 << 20)
	maxAdminUploadSize = int64(30 << 20)
	signedURLTTL       = 15 * time.Minute
)

var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type Service struct {
	db    *gorm.DB
	store storage.Store
}

type DTO struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Role           string  `json:"role,omitempty"`
	MIMEType       string  `json:"mimeType"`
	Size           int64   `json:"size"`
	Width          int     `json:"width,omitempty"`
	Height         int     `json:"height,omitempty"`
	PreviewURL     string  `json:"previewUrl,omitempty"`
	UploadProgress int     `json:"uploadProgress"`
	OwnerID        *string `json:"ownerId,omitempty"`
	CreatedAt      string  `json:"createdAt,omitempty"`
}

func New(db *gorm.DB, store storage.Store) *Service {
	return &Service{db: db, store: store}
}

func (s *Service) UploadUser(
	ctx context.Context,
	userID string,
	header *multipart.FileHeader,
	kind string,
	role string,
) (*DTO, error) {
	if kind != KindSource && kind != KindReference && kind != KindRetouchReference {
		return nil, apierror.Invalid("素材类型无效", nil)
	}
	if kind == KindReference && !validReferenceRole(role) {
		return nil, apierror.Invalid("参考图用途无效", nil)
	}
	if kind != KindReference {
		role = ""
	}
	return s.upload(ctx, &userID, nil, header, kind, role, maxUserUploadSize)
}

func (s *Service) UploadAdminResult(
	ctx context.Context,
	adminID string,
	ownerUserID string,
	header *multipart.FileHeader,
	kind string,
) (*model.Asset, error) {
	if kind != KindRetouchResult {
		return nil, apierror.Invalid("人工交付素材类型无效", nil)
	}
	dto, assetModel, err := s.uploadModel(
		ctx,
		&ownerUserID,
		&adminID,
		header,
		kind,
		"",
		maxAdminUploadSize,
	)
	_ = dto
	return assetModel, err
}

func (s *Service) CreateGenerated(
	ctx context.Context,
	userID string,
	filename string,
	contentType string,
	data []byte,
) (*model.Asset, error) {
	contentType, width, height, err := inspectImage(data, contentType)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxAdminUploadSize {
		return nil, apierror.Invalid("生成图片超过 30MB 限制", nil)
	}

	sum := sha256.Sum256(data)
	objectKey := objectKey(userID, KindAIResult, allowedMIMETypes[contentType])
	if _, err := s.store.Put(ctx, objectKey, bytes.NewReader(data), int64(len(data)), storage.PutOptions{
		ContentType:  contentType,
		CacheControl: "private, max-age=0",
	}); err != nil {
		return nil, fmt.Errorf("store generated asset: %w", err)
	}

	assetModel := model.Asset{
		OwnerUserID: &userID,
		Kind:        KindAIResult,
		OriginalName: safeFilename(
			filename,
			"result"+allowedMIMETypes[contentType],
		),
		MIMEType:  contentType,
		SizeBytes: int64(len(data)),
		Width:     width,
		Height:    height,
		SHA256:    hex.EncodeToString(sum[:]),
		Bucket:    s.store.Bucket(),
		ObjectKey: objectKey,
		Version:   1,
	}
	if err := s.db.WithContext(ctx).Create(&assetModel).Error; err != nil {
		_ = s.store.Delete(ctx, objectKey)
		return nil, err
	}
	return &assetModel, nil
}

func (s *Service) GetOwned(ctx context.Context, userID string, assetID string) (*model.Asset, error) {
	var assetModel model.Asset
	err := s.db.WithContext(ctx).
		Where("id = ? AND owner_user_id = ? AND cleaned_at IS NULL", assetID, userID).
		First(&assetModel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierror.New(http.StatusNotFound, apierror.CodeAssetNotFound, "素材不存在", nil)
	}
	return &assetModel, err
}

func (s *Service) GetByID(ctx context.Context, assetID string) (*model.Asset, error) {
	var assetModel model.Asset
	err := s.db.WithContext(ctx).First(&assetModel, "id = ?", assetID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierror.New(http.StatusNotFound, apierror.CodeAssetNotFound, "素材不存在", nil)
	}
	return &assetModel, err
}

func (s *Service) DTO(ctx context.Context, assetModel model.Asset) (DTO, error) {
	dto := DTO{
		ID:             assetModel.ID,
		Name:           assetModel.OriginalName,
		Kind:           assetModel.Kind,
		Role:           assetModel.ReferenceRole,
		MIMEType:       assetModel.MIMEType,
		Size:           assetModel.SizeBytes,
		Width:          assetModel.Width,
		Height:         assetModel.Height,
		UploadProgress: 100,
		OwnerID:        assetModel.OwnerUserID,
		CreatedAt:      assetModel.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if assetModel.CleanedAt == nil {
		url, err := s.store.PresignGet(ctx, assetModel.ObjectKey, signedURLTTL, storage.PresignOptions{
			ContentType: assetModel.MIMEType,
		})
		if err != nil {
			return DTO{}, err
		}
		dto.PreviewURL = url.String()
	}
	return dto, nil
}

func (s *Service) DownloadURL(ctx context.Context, assetModel model.Asset) (string, error) {
	if assetModel.CleanedAt != nil {
		return "", apierror.New(http.StatusNotFound, apierror.CodeAssetNotFound, "素材不存在", nil)
	}
	url, err := s.store.PresignGet(ctx, assetModel.ObjectKey, signedURLTTL, storage.PresignOptions{
		DownloadName: assetModel.OriginalName,
		ContentType:  assetModel.MIMEType,
	})
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (s *Service) DataURL(ctx context.Context, assetModel model.Asset) (string, error) {
	reader, _, err := s.store.Get(ctx, assetModel.ObjectKey)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	limited := io.LimitReader(reader, maxUserUploadSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxUserUploadSize {
		return "", apierror.Invalid("用于提示词优化的图片超过大小限制", nil)
	}
	return "data:" + assetModel.MIMEType + ";base64," +
		base64.StdEncoding.EncodeToString(data), nil
}

func (s *Service) Open(ctx context.Context, assetModel model.Asset) (io.ReadCloser, error) {
	reader, _, err := s.store.Get(ctx, assetModel.ObjectKey)
	return reader, err
}

func (s *Service) DeleteOwned(ctx context.Context, userID string, assetID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var assetModel model.Asset
		err := tx.Where("id = ? AND owner_user_id = ? AND cleaned_at IS NULL", assetID, userID).
			First(&assetModel).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierror.New(http.StatusNotFound, apierror.CodeAssetNotFound, "素材不存在", nil)
		}
		if err != nil {
			return err
		}
		var relationCount int64
		if err := tx.Model(&model.AssetRelation{}).
			Where("asset_id = ?", assetID).
			Count(&relationCount).Error; err != nil {
			return err
		}
		if relationCount > 0 {
			return apierror.New(
				http.StatusConflict,
				apierror.CodeInvalidInput,
				"素材正在被任务或工单使用，不能删除",
				nil,
			)
		}
		if err := s.store.Delete(ctx, assetModel.ObjectKey); err != nil {
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&assetModel).Updates(map[string]any{
			"cleaned_at":     &now,
			"cleanup_reason": "user_deleted_draft_asset",
			"version":        gorm.Expr("version + 1"),
		}).Error
	})
}

func (s *Service) SetRetained(
	ctx context.Context,
	assetID string,
	adminID string,
	retained bool,
	reason string,
) (*model.Asset, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apierror.Invalid("保留状态变更原因不能为空", nil)
	}
	var value model.Asset
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&value, "id = ?", assetID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apierror.New(http.StatusNotFound, apierror.CodeAssetNotFound, "素材不存在", nil)
			}
			return err
		}
		now := time.Now().UTC()
		updates := map[string]any{
			"retain_permanently": retained,
			"retain_reason":      reason,
			"version":            gorm.Expr("version + 1"),
		}
		if retained {
			updates["retained_by"] = adminID
			updates["retained_at"] = &now
		} else {
			updates["retained_by"] = nil
			updates["retained_at"] = nil
		}
		if err := tx.Model(&value).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&value, "id = ?", assetID).Error
	})
	return &value, err
}

func (s *Service) CleanupManage(ctx context.Context, assetID string, reason string) (*model.Asset, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apierror.Invalid("提前清理原因不能为空", nil)
	}
	var value model.Asset
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&value, "id = ?", assetID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apierror.New(http.StatusNotFound, apierror.CodeAssetNotFound, "素材不存在", nil)
			}
			return err
		}
		if value.CleanedAt != nil {
			return nil
		}
		if value.RetainPermanently {
			return apierror.New(http.StatusConflict, apierror.CodeInvalidInput, "长期保留素材不能清理", nil)
		}
		var relations []model.AssetRelation
		if err := tx.Where("asset_id = ?", assetID).Find(&relations).Error; err != nil {
			return err
		}
		for _, relation := range relations {
			switch relation.ResourceType {
			case "generation_task":
				var count int64
				if err := tx.Model(&model.GenerationTask{}).
					Where("id = ? AND status IN ?", relation.ResourceID, []string{"queued", "processing"}).
					Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return apierror.New(http.StatusConflict, apierror.CodeInvalidInput, "进行中任务正在使用该素材", nil)
				}
			case "retouch_ticket":
				var count int64
				if err := tx.Model(&model.RetouchTicket{}).
					Where("id = ? AND status NOT IN ?", relation.ResourceID, []string{"delivered", "rejected", "cancelled"}).
					Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return apierror.New(http.StatusConflict, apierror.CodeInvalidInput, "进行中人工工单正在使用该素材", nil)
				}
			}
		}
		if err := s.store.Delete(ctx, value.ObjectKey); err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&value).Updates(map[string]any{
			"cleaned_at": &now, "cleanup_reason": reason,
			"version": gorm.Expr("version + 1"),
		}).Error; err != nil {
			return err
		}
		value.CleanedAt = &now
		value.CleanupReason = reason
		return nil
	})
	return &value, err
}

func (s *Service) upload(
	ctx context.Context,
	ownerUserID *string,
	adminID *string,
	header *multipart.FileHeader,
	kind string,
	role string,
	maxBytes int64,
) (*DTO, error) {
	dto, _, err := s.uploadModel(ctx, ownerUserID, adminID, header, kind, role, maxBytes)
	return dto, err
}

func (s *Service) uploadModel(
	ctx context.Context,
	ownerUserID *string,
	adminID *string,
	header *multipart.FileHeader,
	kind string,
	role string,
	maxBytes int64,
) (*DTO, *model.Asset, error) {
	if header == nil || header.Size <= 0 || header.Size > maxBytes {
		return nil, nil, apierror.Invalid(
			fmt.Sprintf("图片大小必须在 1B 到 %dMB 之间", maxBytes>>20),
			nil,
		)
	}
	file, err := header.Open()
	if err != nil {
		return nil, nil, apierror.Invalid("无法读取上传文件", nil)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, nil, apierror.Invalid("图片超过大小限制", nil)
	}
	contentType, width, height, err := inspectImage(data, header.Header.Get("Content-Type"))
	if err != nil {
		return nil, nil, err
	}

	ownerPart := "admin"
	if ownerUserID != nil {
		ownerPart = *ownerUserID
	}
	key := objectKey(ownerPart, kind, allowedMIMETypes[contentType])
	if _, err := s.store.Put(ctx, key, bytes.NewReader(data), int64(len(data)), storage.PutOptions{
		ContentType:  contentType,
		CacheControl: "private, max-age=0",
	}); err != nil {
		return nil, nil, err
	}

	sum := sha256.Sum256(data)
	assetModel := model.Asset{
		OwnerUserID:      ownerUserID,
		CreatedByAdminID: adminID,
		Kind:             kind,
		ReferenceRole:    role,
		OriginalName: safeFilename(
			header.Filename,
			"image"+allowedMIMETypes[contentType],
		),
		MIMEType:  contentType,
		SizeBytes: int64(len(data)),
		Width:     width,
		Height:    height,
		SHA256:    hex.EncodeToString(sum[:]),
		Bucket:    s.store.Bucket(),
		ObjectKey: key,
		Version:   1,
	}
	if err := s.db.WithContext(ctx).Create(&assetModel).Error; err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, nil, err
	}
	dto, err := s.DTO(ctx, assetModel)
	if err != nil {
		return nil, nil, err
	}
	return &dto, &assetModel, nil
}

func inspectImage(data []byte, claimedType string) (string, int, int, error) {
	detected := http.DetectContentType(data)
	if detected == "image/jpg" {
		detected = "image/jpeg"
	}
	if _, allowed := allowedMIMETypes[detected]; !allowed {
		return "", 0, 0, apierror.Invalid("仅支持 JPG、PNG 或 WebP 图片", nil)
	}
	if claimed := strings.TrimSpace(strings.Split(claimedType, ";")[0]); claimed != "" &&
		claimed != "application/octet-stream" &&
		claimed != detected {
		return "", 0, 0, apierror.Invalid("文件扩展信息与图片内容不一致", nil)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", 0, 0, apierror.Invalid("图片文件损坏或无法解码", nil)
	}
	return detected, config.Width, config.Height, nil
}

func validReferenceRole(value string) bool {
	switch value {
	case "style", "composition", "person", "detail":
		return true
	default:
		return false
	}
}

func objectKey(ownerID string, kind string, extension string) string {
	date := time.Now().UTC().Format("2006/01/02")
	return strings.Join([]string{"assets", date, ownerID, kind, uuid.NewString() + extension}, "/")
}

func safeFilename(value string, fallback string) string {
	value = strings.TrimSpace(filepath.Base(strings.ReplaceAll(value, "\\", "/")))
	if value == "" || value == "." {
		return fallback
	}
	runes := []rune(value)
	if len(runes) > 180 {
		return string(runes[:180])
	}
	return value
}
