package dto

import (
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
)

// FaceEventMessage is the Kafka event message published by the Face Recognition (Python) service.
// Received from t_dorm_event topic with snake_case JSON keys.
type FaceEventMessage struct {
	CameraID        string  `json:"camera_id"`
	Building        string  `json:"building"`
	EventType       string  `json:"event_type"`
	StudentID       string  `json:"student_id"`
	Name            string  `json:"name"`
	Confidence      float64 `json:"confidence"`
	Timestamp       int64   `json:"timestamp"`
	FrameSequence   int     `json:"frame_sequence"`
	IsStranger      bool    `json:"is_stranger"`
	SnapshotPath    string  `json:"snapshot_path"`
	DirectionMethod string  `json:"direction_method"`
	X1              float64 `json:"x1"`
	Y1              float64 `json:"y1"`
	X2              float64 `json:"x2"`
	Y2              float64 `json:"y2"`
}

// EventDTO is the API response DTO for an event log entry.
type EventDTO struct {
	ID           int64                `json:"id"`
	CameraID     jsontype.NullString  `json:"camera_id"`
	Building     string               `json:"building"`
	EventType    string               `json:"event_type"`
	StudentID    jsontype.NullString  `json:"student_id"`
	IsStranger   bool                 `json:"is_stranger"`
	Confidence   jsontype.NullFloat64 `json:"confidence"`
	SnapshotPath jsontype.NullString  `json:"snapshot_path"`
	Timestamp    time.Time            `json:"timestamp"`
	CreatedAt    time.Time            `json:"created_at"`
}

// EventQueryDTO is the query parameters for filtering event logs.
type EventQueryDTO struct {
	Building  string     `json:"building" form:"building"`
	CameraID  string     `json:"camera_id" form:"camera_id"`
	EventType string     `json:"event_type" form:"event_type"`
	StudentID string     `json:"student_id" form:"student_id"`
	StartTime *time.Time `json:"start_time" form:"start_time"`
	EndTime   *time.Time `json:"end_time" form:"end_time"`
	Page      int        `json:"page" form:"page"`
	Size      int        `json:"size" form:"size"`
}
