package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/sims/campusvision/dormitory-service-go/internal/handler"
)

// frameMessage is the Kafka message structure from t_dorm_frame topic.
// Produced by stream-gateway with hash-partitioning by building.
type frameMessage struct {
	CameraID      string `json:"camera_id"`
	Building      string `json:"building"`
	FrameSequence int    `json:"frame_sequence"`
	FrameData     string `json:"frame_data"` // base64-encoded JPEG
}

// FrameConsumer consumes t_dorm_frame messages from Kafka and publishes
// decoded frame data to the global FrameHub for WebSocket distribution.
type FrameConsumer struct {
	logger *zap.Logger
	reader *kafka.Reader
	hub    *handler.FrameHub
	cancel context.CancelFunc
}

// NewFrameConsumer creates a new FrameConsumer.
func NewFrameConsumer(
	logger *zap.Logger,
	brokers []string,
	topic string,
	groupID string,
	hub *handler.FrameHub,
) *FrameConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,       // 1 byte
		MaxBytes:    5 << 20, // 5MB (matches t_dorm_frame max message size)
		MaxWait:     1 * time.Second,
	})

	return &FrameConsumer{
		logger: logger,
		reader: reader,
		hub:    hub,
	}
}

// Start launches the consumer loop in a background goroutine.
func (c *FrameConsumer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.consumeLoop(ctx)
	c.logger.Info("Frame consumer started",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group_id", c.reader.Config().GroupID),
	)
}

// Stop gracefully shuts down the consumer.
func (c *FrameConsumer) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.reader.Close()
}

// consumeLoop reads messages from Kafka in a loop until context is cancelled.
func (c *FrameConsumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Frame consumer loop exiting due to context cancellation")
			return
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("Failed to read Kafka message", zap.Error(err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err := c.processMessage(msg); err != nil {
			c.logger.Error("Failed to process frame message",
				zap.Error(err),
				zap.Int64("offset", msg.Offset),
				zap.Int("partition", msg.Partition),
			)
			// Commit anyway to avoid re-processing poison messages
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Warn("Failed to commit Kafka message offset",
				zap.Error(err),
				zap.Int64("offset", msg.Offset),
			)
		}
	}
}

// processMessage deserializes a frame message and publishes it to the FrameHub.
func (c *FrameConsumer) processMessage(msg kafka.Message) error {
	var frame frameMessage
	if err := json.Unmarshal(msg.Value, &frame); err != nil {
		return fmt.Errorf("deserialize frame message: %w", err)
	}

	if frame.CameraID == "" {
		return fmt.Errorf("frame message missing camera_id")
	}
	if frame.FrameData == "" {
		return fmt.Errorf("frame message missing frame_data")
	}

	c.logger.Debug("Received frame",
		zap.String("camera_id", frame.CameraID),
		zap.String("building", frame.Building),
		zap.Int("frame_sequence", frame.FrameSequence),
		zap.Int("data_len", len(frame.FrameData)),
	)

	c.hub.Publish(handler.Frame{
		CameraID:      frame.CameraID,
		FrameData:     frame.FrameData,
		Building:      frame.Building,
		FrameSequence: frame.FrameSequence,
	})

	return nil
}
