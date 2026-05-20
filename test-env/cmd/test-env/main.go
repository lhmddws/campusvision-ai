package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"campusvision/test-env/internal/kafka"
	"campusvision/test-env/internal/server"
	"campusvision/test-env/internal/state"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// ── Env vars ─────────────────────────────────────────────────────────────

	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	frameTopic := getEnv("FRAME_TOPIC", "t_dorm_frame")
	eventTopic := getEnv("EVENT_TOPIC", "t_dorm_event")
	serverPort := getEnv("TEST_SERVER_PORT", "8082")
	mariadbDSN := getEnv("MARIADB_DSN", "sims:sims@tcp(127.0.0.1:3306)/sims?parseTime=true")

	log.Printf("=== CampusVision Test Environment (Go) ===")
	log.Printf("Kafka brokers: %s", kafkaBrokers)
	log.Printf("Frame topic:   %s", frameTopic)
	log.Printf("Event topic:   %s", eventTopic)
	log.Printf("Server port:   %s", serverPort)

	// ── Init state ───────────────────────────────────────────────────────────

	st := state.New()
	st.Config.DBDsn = mariadbDSN

	// ── MariaDB connection ──────────────────────────────────────────────────

	db, err := sql.Open("mysql", mariadbDSN)
	if err != nil {
		log.Printf("[db] MariaDB connection failed: %v", err)
	} else {
		st.MariaDB = db
		log.Println("[db] MariaDB connected")
	}

	useFakeData := getEnv("USE_FAKE_DATA", "true") == "true"
	st.Config.UseFakeData = useFakeData
	log.Printf("USE_FAKE_DATA: %v", useFakeData)

	// Determine base directory for static files and face data.
	// The binary may be invoked from either the repo root or test-env/ directory.
	// We check for the existence of test-env/frontend/dist relative to CWD.
	baseDir := "."
	if _, err := os.Stat("frontend/dist"); os.IsNotExist(err) {
		if _, err := os.Stat("test-env/frontend/dist"); err == nil {
			baseDir = "test-env"
		}
	}

	// ── Init Kafka ──────────────────────────────────────────────────────────

	brokers := strings.Split(kafkaBrokers, ",")
	st.KafkaProducer = kafka.NewWriter(brokers, frameTopic)
	st.KafkaEventPub = kafka.NewWriter(brokers, eventTopic)

	if st.KafkaProducer == nil {
		log.Println("[kafka] Frame producer unavailable — continuing without Kafka")
	}
	if st.KafkaEventPub == nil {
		log.Println("[kafka] Event producer unavailable — continuing without Kafka")
	}

	// Frame producer for webcam → Kafka pipeline
	kafka.InitFrameProducer(brokers, frameTopic)

	// ── Kafka consumer for recognition events ───────────────────────────────

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventConsumer := kafka.NewConsumer(brokers, eventTopic, "test-env-recognition")
	if eventConsumer != nil {
		log.Println("[kafka] Starting consumer for t_dorm_event...")
		go func() {
			kafka.ConsumeEvents(ctx, eventConsumer, func(msg []byte) {
				kafka.HandleRecognitionEvent(msg, st)
			})
		}()
	}

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("[kafka] Shutting down consumer...")
		cancel()
		kafka.CloseConsumer(eventConsumer)
		kafka.CloseFrameProducer()
	}()

	// ── Setup router ─────────────────────────────────────────────────────────

	router := server.SetupRouter(st, baseDir)

	// ── Start server ─────────────────────────────────────────────────────────

	addr := fmt.Sprintf("0.0.0.0:%s", serverPort)
	log.Printf("Server starting on %s", addr)
	log.Printf("Web dashboard : http://localhost:%s/", serverPort)
	log.Printf("API base      : http://localhost:%s/api/", serverPort)
	log.Printf("Health        : http://localhost:%s/api/health", serverPort)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
