package configadapter

import "github.com/ehsanmsb/minigate/internal/domain"

type Config struct {
	Buckets []BucketConfig `yaml:"buckets"`
}

type BucketConfig struct {
	Name      string `yaml:"name"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Path      string `yaml:"path"`
	Endpoint  string `yaml:"endpoint"`
}

func ToDomainBucket(cfg []BucketConfig) map[string]domain.Bucket {
	buckets := make(map[string]domain.Bucket)
	for _, b := range cfg {
		buckets[b.Name] = domain.Bucket{
			Name:      b.Name,
			Bucket:    b.Bucket,
			BasePath:  b.Path,
			AccessKey: b.AccessKey,
			SecretKey: b.SecretKey,
			Endpoint:  b.Endpoint,
		}
	}
	return buckets
}
