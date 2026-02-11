package s3adapter

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
)

type Storage struct {
	clients map[string]*s3.Client
}

func (s *Storage) GetObject(ctx context.Context, bucket string, key string) (io.ReadCloser, error) {
	client, ok := s.clients[bucket]
	if !ok {
		return nil, errors.New("bucket not found")
	}
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}
