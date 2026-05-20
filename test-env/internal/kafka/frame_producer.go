package kafka

import (
	"encoding/base64"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaFrame represents a frame message sent to the t_dorm_frame topic
// from the webcam worker for real-time face recognition.
type KafkaFrame struct {
	CameraID   string `json:"camera_id"`
	Building   string `json:"building"`
	Timestamp  int64  `json:"timestamp"`
	JPEGBase64 string `json:"jpeg_base64"`
}

var frameProducer *kafka.Writer

// InitFrameProducer creates a Kafka writer for frame publishing.
func InitFrameProducer(brokers []string, topic string) {
	frameProducer = NewWriter(brokers, topic)
	if frameProducer != nil {
		log.Printf("[kafka] Frame producer ready for topic %s: %v", topic, brokers)
	}
}

// PushFrame encodes a JPEG frame as base64 and sends to Kafka asynchronously.
func PushFrame(cameraID, building string, jpegData []byte) {
	if frameProducer == nil {
		return
	}
	frame := KafkaFrame{
		CameraID:   cameraID,
		Building:   building,
		Timestamp:  time.Now().UnixMilli(),
		JPEGBase64: base64.StdEncoding.EncodeToString(jpegData),
	}
	if err := SendMessage(frameProducer, building, frame); err != nil {
		log.Printf("[kafka] Frame push warning: %v", err)
	}
}

// CloseFrameProducer safely closes the frame Kafka writer.
func CloseFrameProducer() {
	if frameProducer != nil {
		frameProducer.Close()
		log.Println("[kafka] Frame producer closed")
	}
}
