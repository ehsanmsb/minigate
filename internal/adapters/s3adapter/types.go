package s3adapter

import "github.com/aws/aws-sdk-go-v2/service/s3"

type clientEntry struct {
	client *s3.Client
	bucket string
}
