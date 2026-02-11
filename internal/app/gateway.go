package app

import (
	"context"
	"github.com/ehsanmsb/minigate/internal/domain"
	"github.com/ehsanmsb/minigate/internal/ports"
	"path"
	"strings"
)

type Gateway struct {
	storage ports.ObjectStorage
	buckets map[string]domain.Bucket
}

func NewGateway(storage ports.ObjectStorage, buckets map[string]domain.Bucket) *Gateway {
	return &Gateway{
		storage: storage,
		buckets: buckets,
	}
}

func (g *Gateway) GetObject(ctx context.Context, bucketName string, object string) (*ports.ObjectResult, error) {
	bucket, ok := g.buckets[bucketName]
	if !ok {
		return nil, domain.ErrBucketNotFound
	}

	key := strings.Trim(object, "/")
	base := strings.Trim(bucket.BasePath, "/")
	if base != "" {
		key = path.Join(base, key)
	}

	targetBucket := bucket.Bucket
	if targetBucket == "" {
		targetBucket = bucket.Name
	}
	return g.storage.GetObject(ctx, targetBucket, key)
}
