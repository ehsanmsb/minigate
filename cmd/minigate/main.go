package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ehsanmsb/minigate/internal/config"
	"github.com/ehsanmsb/minigate/internal/gateway"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	addr := flag.String("addr", ":8080", "http listen address")
	metricsAddr := flag.String("metrics-addr", "127.0.0.1:9090", "metrics listen address")
	flag.Parse()
	level := slog.LevelWarn
	if value := os.Getenv("MINIGATE_LOG_LEVEL"); value != "" {
		if err := level.UnmarshalText([]byte(value)); err != nil {
			log.Fatalf("invalid MINIGATE_LOG_LEVEL: %v", err)
		}
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	appHandler, metricsHandler := instrument(gateway.New(cfg))
	server := &http.Server{
		Addr:              *addr,
		Handler:           appHandler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", metricsHandler)
	metricsServer := &http.Server{Addr: *metricsAddr, Handler: metricsMux, ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errorsCh := make(chan error, 2)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errorsCh <- fmt.Errorf("http server: %w", err)
		}
	}()
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errorsCh <- fmt.Errorf("metrics server: %w", err)
		}
	}()
	slog.Info("listening", "addr", *addr, "metrics_addr", *metricsAddr)
	failed := false
	select {
	case err := <-errorsCh:
		slog.Error("serve", "error", err)
		failed = true
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, s := range []*http.Server{server, metricsServer} {
		if err := s.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "error", err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func instrument(handler http.Handler) (http.Handler, http.Handler) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "minigate_http_requests_total",
		Help: "Total HTTP requests by status code and method, excluding health probes.",
	}, []string{"code", "method"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "minigate_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds, excluding health probes.",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	}, []string{"code", "method"})
	registry.MustRegister(requests, duration)
	instrumented := promhttp.InstrumentHandlerCounter(requests, promhttp.InstrumentHandlerDuration(duration, handler))
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			handler.ServeHTTP(w, r)
			return
		}
		instrumented.ServeHTTP(w, r)
	})
	return app, promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}
