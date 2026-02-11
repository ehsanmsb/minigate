package s3adapter

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ehsanmsb/minigate/internal/adapters/configadapter"
)

func NewFromConfig(cfg *configadapter.Config) (*Storage, error) {
	clients := make(map[string]*s3.Client)

	for _, b := range cfg.Buckets {

		awsCfg, err := config.LoadDefaultConfig(
			context.TODO(),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					b.AccessKey,
					b.SecretKey,
					"",
				),
			),
			config.WithEndpointResolverWithOptions(
				aws.EndpointResolverWithOptionsFunc(
					func(service, region string, options ...interface{}) (aws.Endpoint, error) {
						return aws.Endpoint{
							URL:               b.Endpoint,
							HostnameImmutable: true,
						}, nil
					},
				),
			),
			config.WithRegion("us-east-1"),
		)
		if err != nil {
			return nil, err
		}

		clients[b.Name] = s3.NewFromConfig(awsCfg)
	}

	return &Storage{
		clients: clients,
	}, nil
}
