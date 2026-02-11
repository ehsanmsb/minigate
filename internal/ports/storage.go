package ports

import (
	"context"
	"io"
)

type ObjectStorage interface {
	GetObject(ctx context.Context, bucket string, key string) (io.ReadCloser, error)
}
