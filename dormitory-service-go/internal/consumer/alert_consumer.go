package consumer

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// AlertConsumer consumes t_dorm_alert messages from Kafka.
// This is a skeleton implementation for future action routing (acknowledge, notify, etc.).
type AlertConsumer struct {
	logger *zap.Logger
	reader *kafka.Reader
	cancel context.CancelFunc
}

// NewAlertConsumer creates a new AlertConsumer.
func NewAlertConsumer(
	logger *zap.Logger,
	brokers []string,
	topic string,
	groupID string,
) *AlertConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    1 << 20, // 1MB
		MaxWait:     1 * time.Second,
	})

	return &AlertConsumer{
		logger: logger,
		reader: reader,
	}
}

// Start launches the consumer loop in a background goroutine.
func (c *AlertConsumer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.consumeLoop(ctx)
	c.logger.Info("Alert consumer started",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group_id", c.reader.Config().GroupID),
	)
}

// Stop gracefully shuts down the consumer.
func (c *AlertConsumer) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.reader.Close()
}

// consumeLoop reads messages from the t_dorm_alert topic.
// Currently logs all messages; future implementation will route actions
// (e.g., acknowledge alerts, send notifications).
func (c *AlertConsumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Alert consumer loop exiting due to context cancellation")
			return
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("Failed to read alert message", zap.Error(err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		c.logger.Info("Received alert message",
			zap.String("value", string(msg.Value)),
			zap.Int64("offset", msg.Offset),
			zap.Int("partition", msg.Partition),
		)

		// Commit message offset
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Warn("Failed to commit alert message offset",
				zap.Error(err),
				zap.Int64("offset", msg.Offset),
			)
		}
	}
}
