package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// DLQConsumer consumes t_dorm_event_dlq messages from Kafka and logs them.
// This is a skeleton consumer for the Dead Letter Queue — no retry logic.
// Messages in this topic are unrecoverable (poison pills) that failed processing.
type DLQConsumer struct {
	logger *zap.Logger
	reader *kafka.Reader
	cancel context.CancelFunc
}

// NewDLQConsumer creates a new DLQConsumer.
func NewDLQConsumer(
	logger *zap.Logger,
	brokers []string,
	topic string,
	groupID string,
) *DLQConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10 << 20, // 10MB
		MaxWait:     1 * time.Second,
	})

	return &DLQConsumer{
		logger: logger,
		reader: reader,
	}
}

// Start launches the consumer loop in a background goroutine.
func (c *DLQConsumer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.consumeLoop(ctx)
	c.logger.Info("DLQ consumer started",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group_id", c.reader.Config().GroupID),
	)
}

// Stop gracefully shuts down the consumer.
func (c *DLQConsumer) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.reader.Close()
}

// consumeLoop reads messages from Kafka in a loop until context is cancelled.
func (c *DLQConsumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("DLQ consumer loop exiting due to context cancellation")
			return
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("Failed to read DLQ Kafka message", zap.Error(err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		c.processMessage(msg)

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Warn("Failed to commit DLQ message offset",
				zap.Error(err),
				zap.Int64("offset", msg.Offset),
			)
		}
	}
}

// processMessage logs the DLQ message content and offset for manual inspection.
// This is a skeleton — no retry or recovery logic.
func (c *DLQConsumer) processMessage(msg kafka.Message) {
	c.logger.Warn("Received DLQ message",
		zap.Int64("offset", msg.Offset),
		zap.Int("partition", msg.Partition),
		zap.String("key", string(msg.Key)),
		zap.Int("value_size", len(msg.Value)),
		zap.String("topic", msg.Topic),
	)

	// Log first 1KB of message content for inspection (truncate if larger)
	content := string(msg.Value)
	if len(content) > 1024 {
		content = content[:1024] + fmt.Sprintf("... [truncated %d bytes]", len(msg.Value))
	}
	c.logger.Warn("DLQ message content",
		zap.Int64("offset", msg.Offset),
		zap.String("content", content),
	)
}
