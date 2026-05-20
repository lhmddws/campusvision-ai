package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/sims/campusvision/dormitory-service-go/internal/client"
	"github.com/sims/campusvision/dormitory-service-go/internal/config"
	"github.com/sims/campusvision/dormitory-service-go/internal/consumer"
	"github.com/sims/campusvision/dormitory-service-go/internal/handler"
	"github.com/sims/campusvision/dormitory-service-go/internal/middleware"
	redisclient "github.com/sims/campusvision/dormitory-service-go/internal/redis"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"github.com/sims/campusvision/dormitory-service-go/internal/scheduler"
	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

func main() {
	// Load configuration
	cfgPath := "config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		cfgPath = envPath
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Log.Level)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting dormitory-service-go",
		zap.Int("port", cfg.Server.Port),
		zap.String("db_driver", cfg.Database.Driver),
		zap.String("redis_addr", cfg.Redis.Address()),
		zap.Strings("kafka_brokers", cfg.Kafka.Brokers),
	)

	// Connect to MariaDB
	db, err := sqlx.Connect(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.Database.MaxOpenConn)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConn)
	db.SetConnMaxLifetime(10 * time.Minute)

	// Verify database connection
	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}
	logger.Info("Database connection established")

	// Connect to Redis
	rdb := redisclient.NewClient(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.DB,
		cfg.Redis.Password,
	)
	defer rdb.Close()

	if err := rdb.Ping(context.Background()); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	logger.Info("Redis connection established")

	// Initialize repositories
	cameraRepo := repository.NewCameraRepository(db)
	eventLogRepo := repository.NewEventLogRepository(db)
	studentRepo := repository.NewStudentRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	strangerRecordRepo := repository.NewStrangerRecordRepository(db)
	nightlyReportRepo := repository.NewNightlyReportRepository(db)
	configRepo := repository.NewConfigRepository(db)
	cameraLogRepo := repository.NewCameraLogRepository(db)

	// Initialize push client for stream-gateway notifications
	pushBaseURL := os.Getenv("CAMERA_MANAGEMENT_BASE_URL")
	if pushBaseURL == "" {
		pushBaseURL = "http://localhost:8080"
	}
	pushAPIKey := os.Getenv("CAMERA_MANAGEMENT_API_KEY")
	pushClient := client.NewPushClient(pushBaseURL, pushAPIKey)

	// Initialize services
	cameraSvc := service.NewCameraService(cameraRepo, eventLogRepo, cameraLogRepo, pushClient)
	recordSvc := service.NewRecordService(eventLogRepo, studentRepo)
	alertSvc := service.NewAlertService(alertRepo, strangerRecordRepo)
	configSvc := service.NewConfigService(configRepo)
	reportSvc := service.NewReportService(nightlyReportRepo)

	// Initialize handlers
	h := handler.NewHandler(cameraSvc, recordSvc, alertSvc, configSvc, reportSvc, db)
	cameraHandler := handler.NewCameraHandler(cameraSvc)
	recordHandler := handler.NewRecordHandler(recordSvc)
	alertHandler := handler.NewAlertHandler(alertSvc)
	configHandler := handler.NewConfigHandler(configSvc)

	// Initialize Kafka event consumer
	eventConsumer := consumer.NewEventConsumer(
		logger,
		rdb,
		db,
		cfg.Kafka.Brokers,
		cfg.Kafka.EventTopic,
		cfg.Kafka.GroupID,
		cfg.Kafka.MaxPollRecord,
		eventLogRepo,
		studentRepo,
		alertRepo,
		strangerRecordRepo,
		cameraRepo,
	)

	// Initialize Kafka alert consumer
	alertConsumer := consumer.NewAlertConsumer(
		logger,
		cfg.Kafka.Brokers,
		cfg.Kafka.AlertTopic,
		cfg.Kafka.GroupID,
	)

	// Setup consumer manager
	consumerManager := consumer.NewManager(logger)
	consumerManager.Register(eventConsumer)
	consumerManager.Register(alertConsumer)

	// Setup scheduler manager
	schedulerManager := scheduler.NewManager(logger)
	schedulerManager.AddJob("0 0 23 * * *", scheduler.NewNightlyReportJob(logger, reportSvc))
	schedulerManager.AddJob("0 */5 * * * *", scheduler.NewHealthCheckJob(logger, cameraSvc))

	// Setup Gin router
	ginMode := gin.ReleaseMode
	if cfg.Log.Level == "debug" {
		ginMode = gin.DebugMode
	}
	gin.SetMode(ginMode)

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	// Health check endpoint
	router.GET("/api/health", func(c *gin.Context) {
		dbStatus := "ok"
		if err := db.Ping(); err != nil {
			dbStatus = fmt.Sprintf("error: %v", err)
		}

		redisStatus := "ok"
		if err := rdb.Ping(context.Background()); err != nil {
			redisStatus = fmt.Sprintf("error: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "UP",
			"service": "dormitory-service-go",
			"database": gin.H{
				"status": dbStatus,
				"driver": cfg.Database.Driver,
			},
			"redis": gin.H{
				"status": redisStatus,
				"addr":   cfg.Redis.Address(),
			},
			"kafka": gin.H{
				"brokers":     cfg.Kafka.Brokers,
				"event_topic": cfg.Kafka.EventTopic,
				"alert_topic": cfg.Kafka.AlertTopic,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	// Camera routes
	cameras := router.Group("/sims/dorm/cameras")
	{
		cameras.POST("", cameraHandler.RegisterCamera)
		cameras.GET("", cameraHandler.GetCameras)
		cameras.GET("/status", cameraHandler.GetCamerasStatus)
		cameras.GET("/:id", cameraHandler.GetCamera)
		cameras.PUT("/:id", cameraHandler.UpdateCamera)
		cameras.DELETE("/:id", cameraHandler.DeleteCamera)
		cameras.GET("/:id/status", cameraHandler.GetCameraStatus)
		cameras.POST("/:id/health-check", cameraHandler.HealthCheck)
		cameras.GET("/:id/snapshots", cameraHandler.QuerySnapshots)
	}

	// Record routes
	records := router.Group("/sims/dorm/records")
	{
		records.POST("/attendance", recordHandler.HandleAttendance)
		records.GET("/attendance/stats", recordHandler.GetAttendanceStats)
		records.GET("/attendance/daily-summary", recordHandler.GetDailySummary)
		records.GET("/events", recordHandler.GetEvents)
	}

	// Alert routes
	alerts := router.Group("/sims/dorm/alerts")
	{
		alerts.GET("", alertHandler.GetAlerts)
		alerts.POST("/:id/acknowledge", alertHandler.AcknowledgeAlert)
		alerts.GET("/stats", alertHandler.GetAlertStats)
	}

	// Config routes
	configs := router.Group("/api/configs")
	{
		configs.GET("", configHandler.ListConfigs)
		configs.GET("/groups", configHandler.ListGroups)
		configs.GET("/:key", configHandler.GetConfig)
		configs.PUT("/:key", configHandler.UpdateConfig)
		configs.PUT("/batch", configHandler.BatchUpdate)
		configs.POST("/:key/reset", configHandler.ResetConfig)
	}

	// Face routes
	router.POST("/api/face/match", h.FaceMatch)
	router.POST("/api/face/embed", h.FaceEmbed)

	// Start Kafka consumers and schedulers
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	eventConsumer.Start(consumerCtx)
	alertConsumer.Start(consumerCtx)
	schedulerManager.Start()

	logger.Info("Kafka consumers and schedulers started")

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info(fmt.Sprintf("HTTP server listening on :%d", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("Shutting down server...")

	// Stop consumers first
	consumerCancel()
	consumerManager.Stop()

	// Stop schedulers
	schedulerManager.Stop()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

// initLogger creates a zap logger based on the configured level.
func initLogger(level string) (*zap.Logger, error) {
	var lvl zapcore.Level
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Development:      lvl == zapcore.DebugLevel,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return cfg.Build()
}
