package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/sims/campusvision/stream-gateway/internal/config"
)

type FrameMessage struct {
	CameraID       string `json:"camera_id"`
	Building       string `json:"building"`
	Timestamp      int64  `json:"timestamp"`
	FrameSequence  int64  `json:"frame_sequence"`
	FrameData      string `json:"frame_data"`
	FrameWidth     int    `json:"frame_width"`
	FrameHeight    int    `json:"frame_height"`
	JPEGQuality    int    `json:"jpeg_quality"`
	IsDynamic      bool   `json:"is_dynamic"`
}

type Producer struct {
	writer *kafka.Writer
	cfg    config.KafkaConfig
}

func NewProducer(cfg config.KafkaConfig) *Producer {
	compression := compress.Snappy
	if cfg.Compression == "none" {
		compression = compress.None
	} else if cfg.Compression == "gzip" {
		compression = compress.Gzip
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},  // same building → same partition
		Compression:  compression,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: 50 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}

	return &Producer{writer: writer, cfg: cfg}
}

func (p *Producer) SendFrame(ctx context.Context, msg FrameMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal frame message: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.Building),
		Value: data,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
