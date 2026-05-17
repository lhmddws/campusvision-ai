package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sims/campusvision/stream-gateway/internal/camera"
	"github.com/sims/campusvision/stream-gateway/internal/config"
	"github.com/sims/campusvision/stream-gateway/internal/health"
	"github.com/sims/campusvision/stream-gateway/internal/kafka"

	_ "github.com/go-sql-driver/mysql"
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

	// Start DB polling for dynamic camera configuration
	if cfg.Database.DSN != "" {
		go dbPollLoop(ctx, cfg.Database, camManager)
	}

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

func dbPollLoop(ctx context.Context, dbCfg config.DatabaseConfig, camManager *camera.Manager) {
	db, err := sql.Open(dbCfg.Driver, dbCfg.DSN)
	if err != nil {
		log.Printf("[dbpoll] failed to open DB: %v", err)
		return
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Printf("[dbpoll] DB ping failed: %v", err)
	}

	ticker := time.NewTicker(dbCfg.PollInterval)
	defer ticker.Stop()

	syncCamerasFromDB(ctx, db, camManager)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncCamerasFromDB(ctx, db, camManager)
		}
	}
}

func syncCamerasFromDB(ctx context.Context, db *sql.DB, camManager *camera.Manager) {
	rows, err := db.QueryContext(ctx,
		"SELECT camera_id, building, rtsp_url, enabled FROM dorm_camera WHERE enabled = 1")
	if err != nil {
		log.Printf("[dbpoll] query error: %v", err)
		return
	}
	defer rows.Close()

	var cameras []config.CameraConfig
	for rows.Next() {
		var cam config.CameraConfig
		var enabled int
		if err := rows.Scan(&cam.ID, &cam.Building, &cam.RTSPURL, &enabled); err != nil {
			log.Printf("[dbpoll] scan error: %v", err)
			continue
		}
		cam.Enabled = enabled == 1
		cameras = append(cameras, cam)
	}

	camManager.DiffAndSync(cameras)
	log.Printf("[dbpoll] synced %d cameras from DB", len(cameras))
}
