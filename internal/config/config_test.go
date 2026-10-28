package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidatesRoutes(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"duplicate name", "buckets:\n  - name: images\n    endpoint: https://s3.example.com\n    access_key: a\n    secret_key: b\n  - name: images\n    endpoint: https://s3.example.com\n    access_key: a\n    secret_key: b\n", "duplicate bucket name"},
		{"unsafe prefix", "buckets:\n  - name: images\n    endpoint: https://s3.example.com\n    access_key: a\n    secret_key: b\n    path: public/../private\n", "dot segment"},
		{"missing credential", "buckets:\n  - name: images\n    endpoint: https://s3.example.com\n    access_key: a\n", "access_key and secret_key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(filename, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(filename)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
}
