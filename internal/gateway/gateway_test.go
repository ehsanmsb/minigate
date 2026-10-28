package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/ehsanmsb/minigate/internal/config"
)

type fakeStorage struct {
	input *s3.GetObjectInput
	out   *s3.GetObjectOutput
	err   error
	calls int
}

func (f *fakeStorage) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	f.calls++
	f.input = input
	return f.out, f.err
}

func testHandler(f *fakeStorage) *Handler {
	h := New(&config.Config{Buckets: []config.Bucket{{
		Name: "avatars", Bucket: "private-images", Endpoint: "https://storage.example.com",
		Region: "us-east-1", AccessKey: "access", SecretKey: "secret", AllowedPaths: []string{"public"},
	}}})
	r := h.routes["avatars"]
	r.client = f
	h.routes["avatars"] = r
	return h
}

func TestGetObjectUsesConfiguredBucketAndPrefix(t *testing.T) {
	f := &fakeStorage{out: &s3.GetObjectOutput{
		Body:          io.NopCloser(strings.NewReader("image")),
		ContentType:   aws.String("image/png"),
		ContentLength: aws.Int64(5),
	}}
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/avatars/public/photos/a.png", nil))
	if w.Code != http.StatusOK || w.Body.String() != "image" {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
	if got := aws.ToString(f.input.Bucket); got != "private-images" {
		t.Errorf("bucket=%q", got)
	}
	if got := aws.ToString(f.input.Key); got != "public/photos/a.png" {
		t.Errorf("key=%q", got)
	}
	if got := w.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("content type=%q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin=%q", got)
	}
}

func TestGetObjectRejectsPrefixEscape(t *testing.T) {
	for _, url := range []string{"/avatars/public/../secret", "/avatars/public/%2e%2e/secret", "/avatars/private/secret"} {
		f := &fakeStorage{}
		w := httptest.NewRecorder()
		testHandler(f).ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code != http.StatusNotFound || f.calls != 0 {
			t.Errorf("url=%q status=%d calls=%d", url, w.Code, f.calls)
		}
	}
}

func TestAllowedPathsExposeOnlySelectedDirectories(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want int
		key  string
	}{
		{"/avatars/photos/public/a.png", http.StatusOK, "photos/public/a.png"},
		{"/avatars/docs/shared/nested/file.pdf", http.StatusOK, "docs/shared/nested/file.pdf"},
		{"/avatars/photos/private/a.png", http.StatusNotFound, ""},
		{"/avatars/photos/publicity/a.png", http.StatusNotFound, ""},
		{"/avatars/docs/shared-old/file.pdf", http.StatusNotFound, ""},
		{"/avatars/photos/public/../private/a.png", http.StatusNotFound, ""},
	} {
		f := &fakeStorage{out: &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader("ok"))}}
		h := New(&config.Config{Buckets: []config.Bucket{{
			Name: "avatars", Bucket: "private-images", Endpoint: "https://storage.example.com",
			Region: "us-east-1", AccessKey: "access", SecretKey: "secret",
			AllowedPaths: []string{"photos/public", "docs/shared"},
		}}})
		r := h.routes["avatars"]
		r.client = f
		h.routes["avatars"] = r
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.url, nil))
		if w.Code != tc.want {
			t.Errorf("url=%q status=%d want=%d", tc.url, w.Code, tc.want)
		}
		if tc.want == http.StatusOK && (f.calls != 1 || aws.ToString(f.input.Key) != tc.key) {
			t.Errorf("url=%q calls=%d key=%q", tc.url, f.calls, aws.ToString(f.input.Key))
		}
		if tc.want == http.StatusNotFound && f.calls != 0 {
			t.Errorf("url=%q reached storage", tc.url)
		}
	}
}

func TestEncodedPercentIsNotDecodedTwice(t *testing.T) {
	f := &fakeStorage{out: &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader("ok"))}}
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/avatars/public/%252e%252e/file", nil))
	if w.Code != http.StatusOK || aws.ToString(f.input.Key) != "public/%2e%2e/file" {
		t.Errorf("status=%d key=%q", w.Code, aws.ToString(f.input.Key))
	}
}

func TestRouterOnlyAllowsGet(t *testing.T) {
	h := testHandler(&fakeStorage{})
	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/health", http.StatusOK},
		{http.MethodPost, "/avatars/public/file", http.StatusMethodNotAllowed},
		{http.MethodGet, "/missing/file", http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.want {
			t.Errorf("%s %s status=%d want=%d", tc.method, tc.path, w.Code, tc.want)
		}
	}
}

func TestGetObjectRange(t *testing.T) {
	f := &fakeStorage{out: &s3.GetObjectOutput{
		Body:          io.NopCloser(strings.NewReader("data")),
		ContentRange:  aws.String("bytes 0-3/10"),
		ContentLength: aws.Int64(4),
	}}
	request := httptest.NewRequest(http.MethodGet, "/avatars/public/video.mp4", nil)
	request.Header.Set("Range", "bytes=0-3")
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, request)
	if w.Code != http.StatusPartialContent || aws.ToString(f.input.Range) != "bytes=0-3" || w.Header().Get("Content-Range") != "bytes 0-3/10" {
		t.Errorf("status=%d range=%q content-range=%q", w.Code, aws.ToString(f.input.Range), w.Header().Get("Content-Range"))
	}
}

func TestGetObjectErrorStatus(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"missing", &smithy.GenericAPIError{Code: "NoSuchKey"}, http.StatusNotFound},
		{"storage failure", errors.New("connection failed"), http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			testHandler(&fakeStorage{err: tc.err}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/avatars/public/a.png", nil))
			if w.Code != tc.want {
				t.Errorf("status=%d want=%d", w.Code, tc.want)
			}
		})
	}
}

func TestConfiguredEndpointUsesPathStyle(t *testing.T) {
	var upstreamPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "file")
	}))
	defer upstream.Close()

	h := New(&config.Config{Buckets: []config.Bucket{{
		Name: "public", Bucket: "private-files", Endpoint: upstream.URL,
		Region: "us-east-1", AccessKey: "access", SecretKey: "secret",
	}}})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public/folder/file.txt", nil))
	if w.Code != http.StatusOK || w.Body.String() != "file" || upstreamPath != "/private-files/folder/file.txt" {
		t.Fatalf("status=%d body=%q upstream path=%q", w.Code, w.Body.String(), upstreamPath)
	}
}
