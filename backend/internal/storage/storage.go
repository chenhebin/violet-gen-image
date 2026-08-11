package storage

import (
	"context"
	"io"
	"net/url"
	"time"
)

type Store interface {
	Bucket() string
	EnsureBucket(context.Context) error
	Ready(context.Context) error
	Put(context.Context, string, io.Reader, int64, PutOptions) (ObjectInfo, error)
	Get(context.Context, string) (io.ReadCloser, ObjectInfo, error)
	Delete(context.Context, string) error
	PresignGet(context.Context, string, time.Duration, PresignOptions) (*url.URL, error)
}

type PutOptions struct {
	ContentType  string
	CacheControl string
	Metadata     map[string]string
}

type PresignOptions struct {
	DownloadName string
	ContentType  string
}

type ObjectInfo struct {
	Key          string
	Size         int64
	ETag         string
	ContentType  string
	LastModified time.Time
}
