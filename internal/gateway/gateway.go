package gateway

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/ehsanmsb/minigate/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type objectGetter interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type route struct {
	bucket       string
	allowedPaths []string
	client       objectGetter
}

type Handler struct {
	routes map[string]route
	router http.Handler
}

func New(cfg *config.Config) *Handler {
	routes := make(map[string]route, len(cfg.Buckets))
	for _, b := range cfg.Buckets {
		awsCfg := aws.Config{
			Region:      b.Region,
			Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(b.AccessKey, b.SecretKey, "")),
		}
		client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(b.Endpoint)
			o.UsePathStyle = true
		})
		routes[b.Name] = route{bucket: b.Bucket, allowedPaths: b.AllowedPaths, client: client}
	}
	h := &Handler{routes: routes}
	r := chi.NewRouter()
	r.Use(middleware.SetHeader("Access-Control-Allow-Origin", "*"))
	r.Use(middleware.Recoverer)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	r.Get("/{name}/*", h.getObject)
	h.router = r
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) getObject(w http.ResponseWriter, r *http.Request) {
	name, nameErr := routeParam(r, "name")
	key, keyErr := routeParam(r, "*")
	if nameErr != nil || keyErr != nil || name == "" || key == "" || hasDotSegment(key) {
		http.NotFound(w, r)
		return
	}
	selected, ok := h.routes[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if len(selected.allowedPaths) > 0 {
		allowed := false
		for _, prefix := range selected.allowedPaths {
			if strings.HasPrefix(key, prefix+"/") {
				allowed = true
				break
			}
		}
		if !allowed {
			http.NotFound(w, r)
			return
		}
	}
	input := &s3.GetObjectInput{Bucket: aws.String(selected.bucket), Key: aws.String(key)}
	// A full response is safe when we cannot check the client's If-Range validator.
	if value := r.Header.Get("Range"); strings.HasPrefix(value, "bytes=") && r.Header.Get("If-Range") == "" {
		input.Range = aws.String(value)
	}
	obj, err := selected.client.GetObject(r.Context(), input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NoSuchKey", "NotFound":
				http.NotFound(w, r)
				return
			case "AccessDenied":
				// Without ListBucket, S3 also returns AccessDenied for missing keys.
				log.Printf("S3 GetObject denied for route %q: %v", name, err)
				http.NotFound(w, r)
				return
			case "InvalidRange":
				http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
				return
			}
		}
		log.Printf("S3 GetObject failed for route %q: %v", name, err)
		http.Error(w, "storage unavailable", http.StatusBadGateway)
		return
	}
	if obj.Body == nil {
		log.Printf("S3 GetObject returned no body for route %q", name)
		http.Error(w, "storage unavailable", http.StatusBadGateway)
		return
	}
	defer obj.Body.Close()

	copyHeader(w.Header(), "Content-Type", obj.ContentType)
	copyHeader(w.Header(), "Content-Encoding", obj.ContentEncoding)
	copyHeader(w.Header(), "Content-Disposition", obj.ContentDisposition)
	copyHeader(w.Header(), "Cache-Control", obj.CacheControl)
	copyHeader(w.Header(), "ETag", obj.ETag)
	copyHeader(w.Header(), "Content-Range", obj.ContentRange)
	if obj.ContentLength != nil && *obj.ContentLength >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(*obj.ContentLength, 10))
	}
	if obj.LastModified != nil {
		w.Header().Set("Last-Modified", obj.LastModified.UTC().Format(http.TimeFormat))
	}
	w.Header().Set("Accept-Ranges", "bytes")
	status := http.StatusOK
	if obj.ContentRange != nil && *obj.ContentRange != "" {
		status = http.StatusPartialContent
	}
	w.WriteHeader(status)
	if _, err := io.Copy(w, obj.Body); err != nil {
		log.Printf("stream S3 object from route %q: %v", name, err)
	}
}

func routeParam(r *http.Request, name string) (string, error) {
	value := chi.URLParam(r, name)
	if r.URL.RawPath != "" {
		return url.PathUnescape(value)
	}
	return value, nil
}

func copyHeader(h http.Header, name string, value *string) {
	if value != nil && *value != "" {
		h.Set(name, *value)
	}
}

func hasDotSegment(key string) bool {
	for _, segment := range strings.Split(key, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
