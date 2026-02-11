package main

import (
	"flag"
	"github.com/ehsanmsb/minigate/internal/adapters/configadapter"
	"github.com/ehsanmsb/minigate/internal/adapters/httpadapter"
	"github.com/ehsanmsb/minigate/internal/adapters/s3adapter"
	"github.com/ehsanmsb/minigate/internal/app"
	"log"
	"net/http"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	addr := flag.String("addr", ":8080", "http listen address")
	flag.Parse()

	cfg, err := configadapter.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	storage, err := s3adapter.NewFromConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	buckets := configadapter.ToDomainBucket(cfg.Buckets)

	gateway := app.NewGateway(storage, buckets)
	handler := httpadapter.NewGateHandler(gateway)
	server := &http.Server{
		Addr:    *addr,
		Handler: handler.Router(),
	}

	log.Printf("minigate listening on %s", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
