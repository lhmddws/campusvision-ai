package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sims/campusvision/stream-gateway/internal/camera"
	"github.com/sims/campusvision/stream-gateway/internal/config"
	"github.com/sims/campusvision/stream-gateway/internal/health"
	"github.com/sims/campusvision/stream-gateway/internal/kafka"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := kafka.NewProducer(cfg.Kafka)
	defer producer.Close()

	camManager := camera.NewManager(cfg.Frame, cfg.RTSP, producer)
	camManager.Start(ctx, cfg.Cameras)

	mux := http.NewServeMux()
	healthHandler := health.NewHandler(camManager)
	healthHandler.Register(mux)

	healthAddr := cfg.Health.Port
	if healthAddr == 0 {
		healthAddr = 8080
	}

	healthServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", healthAddr),
		Handler: mux,
	}

	go func() {
		log.Printf("Health API listening on :%d", healthAddr)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("health server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	cancel()
	camManager.Stop()
	healthServer.Shutdown(ctx)
}
