package httpadapter

import (
	"fmt"
	"github.com/ehsanmsb/minigate/internal/app"
	"github.com/go-chi/chi/v5"
	"net/http"
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
	object := chi.URLParam(r, "object")
	if bucket == "" || object == "" {
		http.Error(w, "invalid path", http.StatusNotFound)
		return
	}
	fmt.Printf("bucket: %s, object: %s\n", bucket, object)
	fmt.Println(ctx)
}
