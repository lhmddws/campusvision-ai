package server

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"campusvision/test-env/internal/state"
)

func (h *Handler) SSEHandler(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 32)
	h.state.AddSSEClient(ch)
	defer h.state.RemoveSSEClient(ch)
	defer close(ch)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-ch:
			if !ok {
				return false
			}
			fmt.Fprintf(w, "%s\n\n", msg)
			return true
		case <-ticker.C:
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"time\":%d}\n\n", time.Now().UnixMilli())
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

func BroadcastRecognitionSSE(st *state.State, cameraID, eventType, name, studentID string, confidence float64) {
	data, _ := json.Marshal(map[string]interface{}{
		"camera_id":   cameraID,
		"event_type":  eventType,
		"name":        name,
		"student_id":  studentID,
		"confidence":  confidence,
		"timestamp":   time.Now().UnixMilli(),
	})
	event := fmt.Sprintf("event: recognition\ndata: %s", string(data))
	st.BroadcastSSE(event)
}

func BroadcastBehaviorSSE(st *state.State, cameraID, eventType, detail string) {
	data, _ := json.Marshal(map[string]interface{}{
		"camera_id":  cameraID,
		"event_type": eventType,
		"detail":     detail,
		"timestamp":  time.Now().UnixMilli(),
	})
	event := fmt.Sprintf("event: behavior\ndata: %s", string(data))
	st.BroadcastSSE(event)
}
