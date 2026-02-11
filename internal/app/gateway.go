package app

import (
	"github.com/ehsanmsb/minigate/internal/domain"
	"github.com/ehsanmsb/minigate/internal/ports"
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
