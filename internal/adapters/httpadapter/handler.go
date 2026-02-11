package httpadapter

import (
	"bufio"
	"errors"
	"github.com/ehsanmsb/minigate/internal/app"
	"github.com/ehsanmsb/minigate/internal/domain"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
)

type GateHandler struct {
	gateway *app.Gateway
}

func NewGateHandler(gateway *app.Gateway) *GateHandler {
	return &GateHandler{gateway: gateway}
}

func (gh *GateHandler) GetObject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bucket := chi.URLParam(r, "bucket")
	object := chi.URLParam(r, "*")
	if bucket == "" || object == "" {
		http.Error(w, "invalid path", http.StatusNotFound)
		return
	}

	obj, err := gh.gateway.GetObject(ctx, bucket, object)
	if err != nil {
		if errors.Is(err, domain.ErrBucketNotFound) {
			http.Error(w, "bucket not found", http.StatusNotFound)
			return
		}
		http.Error(w, "object not found", http.StatusNotFound)
		return
	}
	defer obj.Body.Close()

	if obj.CacheControl != nil && *obj.CacheControl != "" {
		w.Header().Set("Cache-Control", *obj.CacheControl)
	}
	if obj.ContentType != nil && *obj.ContentType != "" {
		w.Header().Set("Content-Type", *obj.ContentType)
	}
	if obj.ContentLength != nil && *obj.ContentLength > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(*obj.ContentLength, 10))
	}
	if obj.ETag != nil && *obj.ETag != "" {
		w.Header().Set("ETag", *obj.ETag)
	}
	if obj.LastModified != nil {
		w.Header().Set("Last-Modified", obj.LastModified.UTC().Format(http.TimeFormat))
	}

	reader := bufio.NewReader(obj.Body)
	if w.Header().Get("Content-Type") == "" {
		if head, err := reader.Peek(512); err == nil || len(head) > 0 {
			w.Header().Set("Content-Type", http.DetectContentType(head))
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}
