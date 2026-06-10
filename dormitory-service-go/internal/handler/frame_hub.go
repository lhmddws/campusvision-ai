package handler

import (
	"sync"
	"time"
)

// Frame represents a decoded video frame from a camera.
type Frame struct {
	CameraID      string    `json:"camera_id"`
	FrameData     string    `json:"frame_data"` // base64-encoded JPEG
	Building      string    `json:"building"`
	FrameSequence int       `json:"frame_sequence"`
	Timestamp     time.Time `json:"timestamp"`
}

// FrameHub manages WebSocket frame subscribers with camera-level fan-out.
// It is a global singleton that distributes decoded frames from Kafka to
// all subscribers matching a given camera_id.
type FrameHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Frame]struct{} // keyed by camera_id
}

const (
	// frameChannelBuffer is the buffer size for each subscriber channel.
	// If full, the subscriber is skipped (old frame dropped in favor of latest).
	frameChannelBuffer = 3
)

// GlobalFrameHub is the singleton instance used across the application.
var GlobalFrameHub = NewFrameHub()

// NewFrameHub creates a new FrameHub.
func NewFrameHub() *FrameHub {
	return &FrameHub{
		subscribers: make(map[string]map[chan Frame]struct{}),
	}
}

// Subscribe registers a channel for a given camera_id and building.
// Returns the channel (receive-only) and an unsubscribe function.
// The caller must call the unsubscribe function when done to prevent leaks.
func (h *FrameHub) Subscribe(cameraID string, building string) (<-chan Frame, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan Frame, frameChannelBuffer)
	if _, ok := h.subscribers[cameraID]; !ok {
		h.subscribers[cameraID] = make(map[chan Frame]struct{})
	}
	h.subscribers[cameraID][ch] = struct{}{}

	unsubscribe := func() {
		h.Unsubscribe(ch)
	}

	return ch, unsubscribe
}

// Unsubscribe removes a subscriber channel from the hub.
func (h *FrameHub) Unsubscribe(ch chan Frame) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for cameraID, subs := range h.subscribers {
		if _, ok := subs[ch]; ok {
			delete(subs, ch)
			close(ch)
			if len(subs) == 0 {
				delete(h.subscribers, cameraID)
			}
			return
		}
	}
}

// Publish fans out a frame to all subscribers matching the frame's camera_id.
// Non-blocking: if a subscriber's channel is full, the frame is skipped
// (old frame dropped in favor of latest).
func (h *FrameHub) Publish(frame Frame) {
	frame.Timestamp = time.Now()

	h.mu.RLock()
	subs, ok := h.subscribers[frame.CameraID]
	h.mu.RUnlock()

	if !ok || len(subs) == 0 {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// Re-check after acquiring read lock
	subs, ok = h.subscribers[frame.CameraID]
	if !ok {
		return
	}

	for ch := range subs {
		select {
		case ch <- frame:
		default:
			// Channel full — skip (drop old frame, keep latest)
		}
	}
}
