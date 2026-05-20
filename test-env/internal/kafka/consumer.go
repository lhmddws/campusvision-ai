package kafka

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// NewConsumer creates a new Kafka consumer (Reader) for the given topic and consumer group.
// Returns nil if no brokers are configured (graceful degradation).
func NewConsumer(brokers []string, topic string, groupID string) *kafka.Reader {
	if len(brokers) == 0 || brokers[0] == "" {
		log.Println("[kafka] No brokers configured, Kafka consumer disabled")
		return nil
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})

	log.Printf("[kafka] Consumer ready for topic %s (group=%s) at %v", topic, groupID, brokers)
	return r
}

// ConsumeEvents loops forever, fetching messages from the reader and passing them to the handler.
// Returns when the context is cancelled.
func ConsumeEvents(ctx context.Context, reader *kafka.Reader, handler func([]byte)) {
	if reader == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("[kafka] Consumer context cancelled, stopping")
			return
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[kafka] FetchMessage error: %v", err)
				continue
			}

			handler(msg.Value)

			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("[kafka] CommitMessages error: %v", err)
			}
		}
	}
}

// CloseConsumer safely closes a Kafka reader.
func CloseConsumer(reader *kafka.Reader) {
	if reader != nil {
		log.Println("[kafka] Closing consumer...")
		reader.Close()
	}
}
