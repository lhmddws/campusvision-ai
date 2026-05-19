package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"campusvision/test-env/internal/kafka"
	"campusvision/test-env/internal/server"
	"campusvision/test-env/internal/state"
)

func main() {
	// ── Env vars ─────────────────────────────────────────────────────────────

	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	frameTopic := getEnv("FRAME_TOPIC", "t_dorm_frame")
	eventTopic := getEnv("EVENT_TOPIC", "t_dorm_event")
	serverPort := getEnv("TEST_SERVER_PORT", "8082")

	log.Printf("=== CampusVision Test Environment (Go) ===")
	log.Printf("Kafka brokers: %s", kafkaBrokers)
	log.Printf("Frame topic:   %s", frameTopic)
	log.Printf("Event topic:   %s", eventTopic)
	log.Printf("Server port:   %s", serverPort)

	// ── Init state ───────────────────────────────────────────────────────────

	st := state.New()

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
