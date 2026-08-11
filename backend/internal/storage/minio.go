package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"yingyan.local/backend/internal/config"
)

const (
	minPresignDuration = time.Minute
	maxPresignDuration = 24 * time.Hour
)

type MinIOStore struct {
	client        *minio.Client
	presignClient *minio.Client
	bucket        string
	region        string
}

func New(cfg config.StorageConfig) (*MinIOStore, error) {
	endpoint, useSSL, err := normalizeEndpoint(cfg.Endpoint, cfg.UseSSL)
	if err != nil {
		return nil, fmt.Errorf("configure object storage endpoint: %w", err)
	}
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, errors.New("configure object storage: credentials are required")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("configure object storage: bucket is required")
	}
	credential := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credential,
		Secure: useSSL,
		Region: strings.TrimSpace(cfg.Region),
	})
	if err != nil {
		return nil, fmt.Errorf("create object storage client: %w", err)
	}

	presignClient := client
	if publicEndpoint := strings.TrimSpace(cfg.PublicEndpoint); publicEndpoint != "" {
		publicHost, publicSSL, normalizeErr := normalizeEndpoint(publicEndpoint, cfg.UseSSL)
		if normalizeErr != nil {
			return nil, fmt.Errorf("configure public object storage endpoint: %w", normalizeErr)
		}
		if publicHost != endpoint || publicSSL != useSSL {
			presignClient, err = minio.New(publicHost, &minio.Options{
				Creds:  credential,
				Secure: publicSSL,
				Region: strings.TrimSpace(cfg.Region),
			})
			if err != nil {
				return nil, fmt.Errorf("create public object storage client: %w", err)
			}
		}
	}

	return &MinIOStore{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		region:        strings.TrimSpace(cfg.Region),
	}, nil
}

func (s *MinIOStore) Bucket() string {
	return s.bucket
}

// EnsureBucket creates the private bucket when it is absent. MinIO buckets
// are private by default; no public bucket policy is installed here.
func (s *MinIOStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check object storage bucket: %w", err)
	}
	if exists {
		return nil
	}
	err = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region})
	if err == nil {
		return nil
	}
	// Another API or worker process may have created the bucket concurrently.
	exists, checkErr := s.client.BucketExists(ctx, s.bucket)
	if checkErr == nil && exists {
		return nil
	}
	return fmt.Errorf("create object storage bucket: %w", err)
}

func (s *MinIOStore) Ready(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("object storage is not ready: %w", err)
	}
	if !exists {
		return errors.New("object storage is not ready: bucket does not exist")
	}
	return nil
}

func (s *MinIOStore) Put(
	ctx context.Context,
	key string,
	reader io.Reader,
	size int64,
	options PutOptions,
) (ObjectInfo, error) {
	if err := validateObjectKey(key); err != nil {
		return ObjectInfo{}, err
	}
	if reader == nil {
		return ObjectInfo{}, errors.New("put object: reader is required")
	}
	if size < -1 {
		return ObjectInfo{}, errors.New("put object: size must be -1 or greater")
	}
	info, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType:  strings.TrimSpace(options.ContentType),
		CacheControl: strings.TrimSpace(options.CacheControl),
		UserMetadata: cloneMetadata(options.Metadata),
	})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("put object: %w", err)
	}
	return ObjectInfo{
		Key:         info.Key,
		Size:        info.Size,
		ETag:        info.ETag,
		ContentType: strings.TrimSpace(options.ContentType),
	}, nil
}

func (s *MinIOStore) Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, ObjectInfo{}, err
	}
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("get object: %w", err)
	}
	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, ObjectInfo{}, fmt.Errorf("stat object: %w", err)
	}
	return object, ObjectInfo{
		Key:          info.Key,
		Size:         info.Size,
		ETag:         info.ETag,
		ContentType:  info.ContentType,
		LastModified: info.LastModified,
	}, nil
}

func (s *MinIOStore) Delete(ctx context.Context, key string) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (s *MinIOStore) PresignGet(
	ctx context.Context,
	key string,
	expiry time.Duration,
	options PresignOptions,
) (*url.URL, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	if expiry < minPresignDuration || expiry > maxPresignDuration {
		return nil, fmt.Errorf(
			"presign object: expiry must be between %s and %s",
			minPresignDuration,
			maxPresignDuration,
		)
	}
	parameters := make(url.Values)
	if downloadName := strings.TrimSpace(options.DownloadName); downloadName != "" {
		parameters.Set("response-content-disposition", contentDisposition(downloadName))
	}
	if contentType := strings.TrimSpace(options.ContentType); contentType != "" {
		parameters.Set("response-content-type", contentType)
	}
	signedURL, err := s.presignClient.PresignedGetObject(ctx, s.bucket, key, expiry, parameters)
	if err != nil {
		return nil, fmt.Errorf("presign object: %w", err)
	}
	return signedURL, nil
}

func normalizeEndpoint(rawEndpoint string, defaultSSL bool) (string, bool, error) {
	value := strings.TrimSpace(rawEndpoint)
	if value == "" {
		return "", false, errors.New("endpoint is required")
	}
	if !strings.Contains(value, "://") {
		if strings.ContainsAny(value, "/?#") {
			return "", false, errors.New("endpoint must be host:port without a path")
		}
		return value, defaultSSL, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", false, errors.New("endpoint is invalid")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false, errors.New("endpoint must not contain credentials, query, or fragment")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", false, errors.New("endpoint must not contain a path")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return parsed.Host, true, nil
	case "http":
		return parsed.Host, false, nil
	default:
		return "", false, errors.New("endpoint must use http or https")
	}
}

func validateObjectKey(key string) error {
	if key == "" {
		return errors.New("object key is required")
	}
	if len(key) > 1024 {
		return errors.New("object key is too long")
	}
	if strings.HasPrefix(key, "/") ||
		strings.Contains(key, "\\") ||
		path.Clean(key) != key ||
		hasParentSegment(key) {
		return errors.New("object key is not canonical")
	}
	for _, character := range key {
		if unicode.IsControl(character) {
			return errors.New("object key contains control characters")
		}
	}
	return nil
}

func hasParentSegment(key string) bool {
	for _, segment := range strings.Split(key, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func contentDisposition(filename string) string {
	// mime.FormatMediaType safely quotes ASCII names and emits RFC 2231 for
	// non-ASCII names, preventing response-header injection.
	value := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if value == "" {
		return "attachment"
	}
	return value
}

var _ Store = (*MinIOStore)(nil)
