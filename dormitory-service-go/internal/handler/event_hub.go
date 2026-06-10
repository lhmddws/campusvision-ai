package handler

import (
	"sync"
	"time"
)

// RecognitionEvent represents a face recognition event for SSE streaming.
type RecognitionEvent struct {
	CameraID      string    `json:"camera_id"`
	X1            float64   `json:"x1"`
	Y1            float64   `json:"y1"`
	X2            float64   `json:"x2"`
	Y2            float64   `json:"y2"`
	Name          string    `json:"name"`
	Confidence    float64   `json:"confidence"`
	FrameSequence int       `json:"frame_sequence"`
	EventType     string    `json:"event_type"`
	Timestamp     time.Time `json:"timestamp"`
}

// EventHub manages SSE subscribers for recognition event broadcasting.
type EventHub struct {
	mu          sync.RWMutex
	subscribers map[chan RecognitionEvent]struct{}
}

const eventChannelBuffer = 64

// GlobalEventHub is the singleton instance used across the application.
var GlobalEventHub = NewEventHub()

// NewEventHub creates a new EventHub.
func NewEventHub() *EventHub {
	return &EventHub{
		subscribers: make(map[chan RecognitionEvent]struct{}),
	}
}

// Subscribe registers a channel for receiving recognition events.
// Returns the channel (receive-only) and an unsubscribe function.
// The caller must call the unsubscribe function when done to prevent leaks.
func (h *EventHub) Subscribe() (<-chan RecognitionEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan RecognitionEvent, eventChannelBuffer)
	h.subscribers[ch] = struct{}{}

	unsubscribe := func() {
		h.Unsubscribe(ch)
	}

	return ch, unsubscribe
}

// Unsubscribe removes a subscriber channel from the hub.
func (h *EventHub) Unsubscribe(ch chan RecognitionEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
}

// Publish fans out a recognition event to all subscribers.
// Non-blocking: if a subscriber's channel is full, the event is skipped
// (old event dropped in favor of latest).
func (h *EventHub) Publish(event RecognitionEvent) {
	event.Timestamp = time.Now()

	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full — skip (drop old event)
		}
	}
}
