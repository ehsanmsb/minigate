package config

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Buckets []Bucket `yaml:"buckets"`
}

type Bucket struct {
	Name         string   `yaml:"name"`
	Bucket       string   `yaml:"bucket"`
	Endpoint     string   `yaml:"endpoint"`
	Region       string   `yaml:"region"`
	AccessKey    string   `yaml:"access_key"`
	SecretKey    string   `yaml:"secret_key"`
	AllowedPaths []string `yaml:"allowed_paths"`
}

func Load(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("config contains multiple YAML documents")
		}
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if len(cfg.Buckets) == 0 {
		return nil, fmt.Errorf("config must contain at least one bucket")
	}
	seen := make(map[string]bool, len(cfg.Buckets))
	for i := range cfg.Buckets {
		b := &cfg.Buckets[i]
		if b.Name == "" || b.Name == "." || b.Name == ".." || strings.ContainsAny(b.Name, "/\\") {
			return nil, fmt.Errorf("bucket %d: invalid name %q", i+1, b.Name)
		}
		if seen[b.Name] {
			return nil, fmt.Errorf("duplicate bucket name %q", b.Name)
		}
		seen[b.Name] = true
		if b.Bucket == "" {
			b.Bucket = b.Name
		}
		u, err := url.Parse(b.Endpoint)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("bucket %q: invalid endpoint %q", b.Name, b.Endpoint)
		}
		if b.AccessKey == "" || b.SecretKey == "" {
			return nil, fmt.Errorf("bucket %q: access_key and secret_key are required", b.Name)
		}
		if b.Region == "" {
			b.Region = "us-east-1"
		}
		if b.AllowedPaths != nil && len(b.AllowedPaths) == 0 {
			return nil, fmt.Errorf("bucket %q: allowed_paths cannot be empty", b.Name)
		}
		for j, allowed := range b.AllowedPaths {
			allowed = strings.Trim(allowed, "/")
			if allowed == "" || hasDotSegment(allowed) {
				return nil, fmt.Errorf("bucket %q: invalid allowed path %q", b.Name, b.AllowedPaths[j])
			}
			b.AllowedPaths[j] = allowed
		}
	}
	return &cfg, nil
}

func hasDotSegment(key string) bool {
	for _, segment := range strings.Split(key, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
