package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsCountRequestsWithoutHealthProbes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	app, metrics := instrument(handler, 2)
	for _, path := range []string{"/health", "/file", "/missing"} {
		app.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	w := httptest.NewRecorder()
	metrics.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("metrics status=%d", w.Code)
	}
	for _, sample := range []string{
		`minigate_configured_buckets 2`,
		`minigate_http_requests_total{code="200",method="get"} 1`,
		`minigate_http_requests_total{code="404",method="get"} 1`,
		`minigate_http_request_duration_seconds_count{code="200",method="get"} 1`,
		`minigate_http_request_duration_seconds_count{code="404",method="get"} 1`,
		`go_goroutines`,
	} {
		if !strings.Contains(w.Body.String(), sample) {
			t.Errorf("missing metric sample %q", sample)
		}
	}
	if strings.Contains(w.Body.String(), `path=`) {
		t.Fatal("object paths must not be metric labels")
	}
}
