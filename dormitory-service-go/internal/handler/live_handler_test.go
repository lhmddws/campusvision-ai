package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testJWTSecret = "test-secret-for-unittests"

// generateTestToken creates a valid JWT token for testing.
func generateTestToken(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":      "1",
		"username": "admin",
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

// ---------------------------------------------------------------------------
// WebSocket tests
// ---------------------------------------------------------------------------

func TestLiveHandler_WebSocket_InvalidToken(t *testing.T) {
	hub := NewFrameHub()
	handler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/ws/live?camera_id=cam1&token=badtoken", nil)

	handler.HandleWebSocket(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(401), resp["code"])
	assert.Equal(t, "unauthorized", resp["msg"])
}

func TestLiveHandler_WebSocket_MissingToken(t *testing.T) {
	hub := NewFrameHub()
	handler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/ws/live?camera_id=cam1", nil)

	handler.HandleWebSocket(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(401), resp["code"])
}

func TestLiveHandler_WebSocket_ValidConnection(t *testing.T) {
	hub := NewFrameHub()
	handler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	router := gin.New()
	router.GET("/ws/live", handler.HandleWebSocket)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	token := generateTestToken(t, testJWTSecret)
	url := fmt.Sprintf("ws://%s/ws/live?camera_id=cam1&token=%s",
		server.Listener.Addr().String(), token)

	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err, "WebSocket upgrade should succeed with valid token")

	// Verify 101 Switching Protocols
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode,
		"expected 101 Switching Protocols on successful WebSocket upgrade")

	// Clean close
	_ = conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	conn.Close()
}

func TestLiveHandler_WebSocket_ReceivesFrame(t *testing.T) {
	hub := NewFrameHub()
	handler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	router := gin.New()
	router.GET("/ws/live", handler.HandleWebSocket)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	token := generateTestToken(t, testJWTSecret)
	url := fmt.Sprintf("ws://%s/ws/live?camera_id=cam1&token=%s",
		server.Listener.Addr().String(), token)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	// Allow handler goroutine time to subscribe to the hub
	time.Sleep(50 * time.Millisecond)

	// Publish a frame — handler's write pump should deliver it over WebSocket
	frame := Frame{
		CameraID:      "cam1",
		FrameData:     "dGVzdCBmcmFtZSBkYXRh", // base64 "test frame data"
		Building:      "A",
		FrameSequence: 42,
	}
	hub.Publish(frame)

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err, "should receive a frame message over WebSocket")

	var received Frame
	err = json.Unmarshal(msg, &received)
	require.NoError(t, err, "frame should be valid JSON")

	assert.Equal(t, "cam1", received.CameraID)
	assert.Equal(t, "dGVzdCBmcmFtZSBkYXRh", received.FrameData)
	assert.Equal(t, "A", received.Building)
	assert.Equal(t, 42, received.FrameSequence)
	assert.False(t, received.Timestamp.IsZero(), "timestamp should be set by hub")
}

// ---------------------------------------------------------------------------
// SSE tests
// ---------------------------------------------------------------------------

func TestLiveHandler_SSE_InvalidToken(t *testing.T) {
	hub := NewFrameHub()
	handler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sse/live?token=badtoken", nil)

	handler.HandleSSE(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(401), resp["code"])
	assert.Equal(t, "unauthorized", resp["msg"])
}

func TestLiveHandler_SSE_ValidConnection(t *testing.T) {
	hub := NewFrameHub()
	handler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	token := generateTestToken(t, testJWTSecret)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sse/live?token="+token, nil)

	// Use a cancellable context so we can stop the SSE streaming loop
	ctx, cancel := context.WithCancel(context.Background())
	c.Request = c.Request.WithContext(ctx)

	// Run the SSE handler in a goroutine; it sets headers then enters an
	// infinite keepalive loop.  We give it a short window to set headers
	// before checking them.
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.HandleSSE(c)
	}()

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

	cancel()
	<-done // wait for handler to exit
}

func TestLiveHandler_SSE_Keepalive(t *testing.T) {
	hub := NewFrameHub()
	handler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	router := gin.New()
	router.GET("/sse/live", handler.HandleSSE)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	token := generateTestToken(t, testJWTSecret)

	resp, err := http.Get(server.URL + "/sse/live?token=" + token)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	// Headers are available immediately
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Verify keepalive comments are sent. The handler sends a keepalive
	// comment line (" : keepalive") every 15 seconds.  We read the
	// streaming body until we find it or hit a deadline.
	readDone := make(chan struct{})
	var buf bytes.Buffer

	go func() {
		defer close(readDone)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			buf.WriteString(line + "\n")
			if strings.Contains(line, "keepalive") {
				return
			}
		}
	}()

	select {
	case <-readDone:
		assert.Contains(t, buf.String(), "keepalive",
			"SSE stream should contain at least one keepalive comment")
	case <-time.After(16 * time.Second):
		// The handler uses a hardcoded 15-second ticker; 16s is enough
		// for the first tick to fire.
		t.Fatal("timed out waiting for keepalive comment (handler uses 15s interval)")
	}
}

func TestLiveHandler_SSE_ReceivesRecognitionEvent(t *testing.T) {
	hub := NewFrameHub()
	handler := NewLiveHandler(hub, testJWTSecret, zap.NewNop())

	router := gin.New()
	router.GET("/sse/live", handler.HandleSSE)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	token := generateTestToken(t, testJWTSecret)

	resp, err := http.Get(server.URL + "/sse/live?token=" + token)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Publish a recognition event to the global hub
	event := RecognitionEvent{
		CameraID:      "cam1",
		X1:            10.5,
		Y1:            20.3,
		X2:            100.7,
		Y2:            200.1,
		Name:          "Test Student",
		Confidence:    0.95,
		FrameSequence: 42,
		EventType:     "entry",
	}
	GlobalEventHub.Publish(event)

	// Read the SSE stream until we find the "data:" line (the event payload)
	readDone := make(chan struct{})
	var buf bytes.Buffer

	go func() {
		defer close(readDone)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			buf.WriteString(line + "\n")
			if strings.HasPrefix(line, "data:") {
				return
			}
		}
	}()

	select {
	case <-readDone:
		body := buf.String()
		assert.Contains(t, body, "event: recognition",
			"SSE stream should contain 'event: recognition' line")
		assert.Contains(t, body, `"camera_id":"cam1"`,
			"SSE data should include camera_id")
		assert.Contains(t, body, `"name":"Test Student"`,
			"SSE data should include name")
		assert.Contains(t, body, `"confidence":0.95`,
			"SSE data should include confidence")
		assert.Contains(t, body, `"x1":10.5`,
			"SSE data should include x1 bounding box coordinate")
		assert.Contains(t, body, `"frame_sequence":42`,
			"SSE data should include frame_sequence")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for recognition event")
	}
}
