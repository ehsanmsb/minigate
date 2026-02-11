package ports

import (
	"context"
	"io"
	"time"
)

type ObjectResult struct {
	Body          io.ReadCloser
	ContentType   *string
	ContentLength *int64
	ETag          *string
	LastModified  *time.Time
	CacheControl  *string
}

type ObjectStorage interface {
	GetObject(ctx context.Context, bucket string, key string) (*ObjectResult, error)
}
