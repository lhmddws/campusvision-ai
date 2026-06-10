package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Integration tests for live preview pipeline: FrameHub → WebSocket / SSE.
//
// These tests validate the full in-process pipeline without requiring Kafka.
// Frames are published directly to FrameHub, simulating what FrameConsumer does.
// FrameConsumer itself is not involved—only the FrameHub → WebSocket/SSE path.

// ---------------------------------------------------------------------------
// Test 1: WebSocket receives a frame published through FrameHub
// ---------------------------------------------------------------------------

func TestWebSocketReceivesFrame(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewFrameHub()
	liveHandler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	router := gin.New()
	router.GET("/ws/live", liveHandler.HandleWebSocket)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	token := generateTestToken(t, testJWTSecret)
	url := fmt.Sprintf("ws://%s/ws/live?camera_id=cam1&token=%s",
		server.Listener.Addr().String(), token)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err, "WebSocket upgrade should succeed with valid token")
	t.Cleanup(func() { conn.Close() })

	// Allow the handler's write pump goroutine to subscribe to the hub
	time.Sleep(50 * time.Millisecond)

	// Publish a frame — simulates what FrameConsumer does after Kafka message
	frame := Frame{
		CameraID:      "cam1",
		FrameData:     "aW50ZWdyYXRpb24tZnJhbWUtZGF0YQ==",
		Building:      "A",
		FrameSequence: 42,
	}
	hub.Publish(frame)

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err, "should receive a frame message over WebSocket")

	var received Frame
	err = json.Unmarshal(msg, &received)
	require.NoError(t, err, "frame message should be valid JSON")

	assert.Equal(t, "cam1", received.CameraID)
	assert.Equal(t, "aW50ZWdyYXRpb24tZnJhbWUtZGF0YQ==", received.FrameData)
	assert.Equal(t, "A", received.Building)
	assert.Equal(t, 42, received.FrameSequence)
	assert.False(t, received.Timestamp.IsZero(), "timestamp should be populated by FrameHub.Publish")
}

// ---------------------------------------------------------------------------
// Test 2: SSE endpoint connects with correct headers
// ---------------------------------------------------------------------------

func TestSSEConnects(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewFrameHub()
	liveHandler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	token := generateTestToken(t, testJWTSecret)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sse/live?token="+token, nil)

	// Provide a cancellable context so we can stop the SSE keepalive loop
	ctx, cancel := context.WithCancel(context.Background())
	c.Request = c.Request.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		liveHandler.HandleSSE(c)
	}()

	// Give the handler time to set headers before checking
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, http.StatusOK, w.Code, "SSE handler should respond with 200 OK")
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

	cancel()
	<-done // wait for handler to exit cleanly
}

// ---------------------------------------------------------------------------
// Test 3: WebSocket rejects invalid tokens
// ---------------------------------------------------------------------------

func TestWebSocketInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewFrameHub()
	liveHandler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	router := gin.New()
	router.GET("/ws/live", liveHandler.HandleWebSocket)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	url := fmt.Sprintf("ws://%s/ws/live?camera_id=cam1&token=invalid-token-bogus",
		server.Listener.Addr().String())

	_, resp, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		// Expected: gorilla/websocket returns an error for non-101 responses.
		// The HTTP response should carry the 401 status set by the handler.
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
				"handler should reject invalid token with 401")
		}
	} else {
		t.Error("expected WebSocket dial to fail with invalid token")
	}
}

// ---------------------------------------------------------------------------
// Test 4: Multiple WebSocket connections receive the same published frame
// ---------------------------------------------------------------------------

func TestMultipleWebSocketConnections(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewFrameHub()
	liveHandler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	router := gin.New()
	router.GET("/ws/live", liveHandler.HandleWebSocket)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	token := generateTestToken(t, testJWTSecret)
	baseURL := fmt.Sprintf("ws://%s/ws/live?camera_id=cam2&token=%s",
		server.Listener.Addr().String(), token)

	// Connect two clients to the same camera
	conn1, _, err := websocket.DefaultDialer.Dial(baseURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn1.Close() })

	conn2, _, err := websocket.DefaultDialer.Dial(baseURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn2.Close() })

	time.Sleep(50 * time.Millisecond) // let both subscriptions register

	// Publish a single frame — should fan out to both clients via FrameHub
	frame := Frame{
		CameraID:      "cam2",
		FrameData:     "bXVsdGktY2xpZW50LWZyYW1l",
		Building:      "B",
		FrameSequence: 99,
	}
	hub.Publish(frame)

	_ = conn1.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg1, err := conn1.ReadMessage()
	require.NoError(t, err, "client 1 should receive the frame")

	_ = conn2.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg2, err := conn2.ReadMessage()
	require.NoError(t, err, "client 2 should receive the frame")

	var f1, f2 Frame
	require.NoError(t, json.Unmarshal(msg1, &f1))
	require.NoError(t, json.Unmarshal(msg2, &f2))

	// Both should have identical frame content (timestamp may differ by nanoseconds)
	assert.Equal(t, f1.CameraID, f2.CameraID, "both clients should see same camera_id")
	assert.Equal(t, f1.FrameData, f2.FrameData, "both clients should see same frame_data")
	assert.Equal(t, f1.FrameSequence, f2.FrameSequence, "both clients should see same frame_sequence")
	assert.Equal(t, f1.Building, f2.Building, "both clients should see same building")
	assert.NotZero(t, f1.Timestamp, "client 1 timestamp should be set")
	assert.NotZero(t, f2.Timestamp, "client 2 timestamp should be set")
}

// ---------------------------------------------------------------------------
// Test 5: Client disconnect unsubscribes; disconnected client no longer
// receives frames, and new clients continue to work normally
// ---------------------------------------------------------------------------

func TestWebSocketUnsubscribe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewFrameHub()
	liveHandler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	router := gin.New()
	router.GET("/ws/live", liveHandler.HandleWebSocket)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	token := generateTestToken(t, testJWTSecret)
	baseURL := fmt.Sprintf("ws://%s/ws/live?camera_id=cam3&token=%s",
		server.Listener.Addr().String(), token)

	// -- Step 1: Connect a client, receive a frame --
	conn1, _, err := websocket.DefaultDialer.Dial(baseURL, nil)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	frame1 := Frame{
		CameraID:      "cam3",
		FrameData:     "Zmlyc3QtZnJhbWU=",
		Building:      "C",
		FrameSequence: 1,
	}
	hub.Publish(frame1)

	_ = conn1.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = conn1.ReadMessage()
	require.NoError(t, err, "client should receive the first frame")

	// -- Step 2: Gracefully disconnect the client --
	err = conn1.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)
	conn1.Close()
	time.Sleep(150 * time.Millisecond) // allow handler's read pump to detect disconnect and unsubscribe

	// -- Step 3: Publish another frame; ensure no panic occurs (the hub
	//    should have removed the subscription) --
	require.NotPanics(t, func() {
		hub.Publish(Frame{
			CameraID:      "cam3",
			FrameData:     "c2Vjb25kLWZyYW1l",
			Building:      "C",
			FrameSequence: 2,
		})
	}, "publishing after all subscribers disconnect should not panic")

	// -- Step 4: Connect a fresh client, subscribe to the same camera,
	//    publish a new frame, and verify the new client receives it --
	conn2, _, err := websocket.DefaultDialer.Dial(baseURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn2.Close() })
	time.Sleep(50 * time.Millisecond)

	frame3 := Frame{
		CameraID:      "cam3",
		FrameData:     "dGhpcmQtZnJhbWU=",
		Building:      "C",
		FrameSequence: 3,
	}
	hub.Publish(frame3)

	_ = conn2.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg2, err := conn2.ReadMessage()
	require.NoError(t, err, "new client should receive a frame")

	var received Frame
	require.NoError(t, json.Unmarshal(msg2, &received))
	assert.Equal(t, 3, received.FrameSequence,
		"new client should receive frame 3 (not the frame published while unsubscribed)")
	assert.Equal(t, "dGhpcmQtZnJhbWU=", received.FrameData)
	assert.Equal(t, "cam3", received.CameraID)
}
