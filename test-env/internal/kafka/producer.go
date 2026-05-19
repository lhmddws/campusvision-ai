package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// NewWriter creates a new Kafka writer with the given brokers and topic.
// Returns nil if connection fails (graceful degradation).
func NewWriter(brokers []string, topic string) *kafka.Writer {
	if len(brokers) == 0 || brokers[0] == "" {
		log.Println("[kafka] No brokers configured, Kafka disabled")
		return nil
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		BatchTimeout: 50 * time.Millisecond,
		RequiredAcks: kafka.RequireNone, // don't wait for ack - best effort
		Async:        true,               // non-blocking writes
	}

	// Test connectivity briefly
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		log.Printf("[kafka] Cannot connect to %s for topic %s: %v — Kafka disabled", brokers[0], topic, err)
		return nil
	}
	conn.Close()

	log.Printf("[kafka] Producer ready for topic %s at %v", topic, brokers)
	return w
}

// SendMessage publishes a JSON-serializable message to the Kafka writer.
// Returns an error if the writer is nil or send fails.
func SendMessage(writer *kafka.Writer, key string, msg interface{}) error {
	if writer == nil {
		return fmt.Errorf("kafka writer is nil")
	}

	value, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}
	return nil
}

// CloseWriter safely closes a Kafka writer.
func CloseWriter(writer *kafka.Writer) {
	if writer != nil {
		writer.Close()
	}
}
