package kafka

import (
	"encoding/json"
	"fmt"
	"log"

	"campusvision/test-env/internal/state"
)

// RecognitionEvent matches the t_dorm_event JSON format from face-recognition Python.
type RecognitionEvent struct {
	CameraID   string  `json:"camera_id"`
	EventType  string  `json:"event_type"`
	StudentID  string  `json:"student_id"`
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	IsStranger bool    `json:"is_stranger"`
	Timestamp  int64   `json:"timestamp"`
	Source     string  `json:"source"`
	Detail     string  `json:"detail"`
}

// HandleRecognitionEvent parses a t_dorm_event message and updates the shared state.
func HandleRecognitionEvent(data []byte, st *state.State) {
	var ev RecognitionEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		log.Printf("[kafka] Failed to parse event: %v", err)
		return
	}

	detail := fmt.Sprintf("识别: %s %s (%.0f%%)", ev.Name, ev.EventType, ev.Confidence*100)
	if ev.Source == "behavior" {
		detail = ev.Detail
		ev.EventType = "behavior"
	}

	st.LogEvent(ev.CameraID, ev.EventType, detail)
	st.StoreRecognitionResult(ev.CameraID, state.RecognitionResult{
		CameraID:   ev.CameraID,
		EventType:  ev.EventType,
		StudentID:  ev.StudentID,
		Name:       ev.Name,
		Confidence: ev.Confidence,
		IsStranger: ev.IsStranger,
		Timestamp:  ev.Timestamp,
	})

	st.BroadcastSSE(formatSSEEvent(&ev))

	log.Printf("[kafka] recognition: %s, %s, %s (%.2f)", ev.CameraID, ev.EventType, ev.Name, ev.Confidence)
}

func formatSSEEvent(ev *RecognitionEvent) string {
	data, _ := json.Marshal(map[string]interface{}{
		"camera_id":   ev.CameraID,
		"event_type":  ev.EventType,
		"name":        ev.Name,
		"student_id":  ev.StudentID,
		"confidence":  ev.Confidence,
		"timestamp":   ev.Timestamp,
	})
	if ev.Source == "behavior" {
		return fmt.Sprintf("event: behavior\ndata: %s", string(data))
	}
	return fmt.Sprintf("event: recognition\ndata: %s", string(data))
}
