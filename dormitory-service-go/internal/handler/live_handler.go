package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// LiveHandler handles WebSocket and SSE endpoints for live preview streaming.
type LiveHandler struct {
	frameHub  *FrameHub
	hub       *FrameHub
	jwtSecret string
	logger    *zap.Logger
}

// NewLiveHandler creates a new LiveHandler with the given FrameHub and logger.
func NewLiveHandler(hub *FrameHub, jwtSecret string, logger *zap.Logger) *LiveHandler {
	return &LiveHandler{
		frameHub:  hub,
		hub:       hub,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// validateToken checks whether the provided JWT token string is valid.
func (h *LiveHandler) validateToken(tokenString string) bool {
	if tokenString == "" {
		return false
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})

	return err == nil && token.Valid
}

// HandleWebSocket handles GET /ws/live?token=JWT&camera_id=xxx
//
// It validates the JWT token from the query parameter, upgrades the connection
// to WebSocket, subscribes to FrameHub, and streams JSON-encoded frames to
// the client. Client disconnect is detected by a blocking read loop.
func (h *LiveHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if !h.validateToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "unauthorized"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	cameraID := c.DefaultQuery("camera_id", "")
	subCh, unsubscribe := h.frameHub.Subscribe(cameraID, "")
	defer unsubscribe()

	// Write pump: reads frames from FrameHub subscription and writes to WebSocket.
	// The goroutine exits when subCh is closed (via unsubscribe in defer).
	go func() {
		defer func() {
			_ = conn.Close()
		}()

		for frame := range subCh {
			msg, err := json.Marshal(frame)
			if err != nil {
				h.logger.Error("Failed to marshal frame", zap.Error(err))
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				h.logger.Warn("WebSocket write error", zap.Error(err))
				return
			}
		}
	}()

	// Read pump: blocks until the client disconnects or an error occurs.
	// On disconnect, the defer runs and unsubscribes, closing subCh and
	// causing the write pump to exit gracefully.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// HandleSSE handles GET /sse/live?token=JWT
//
// It validates the JWT token from the query parameter, sets SSE headers, and
// streams recognition events from GlobalEventHub as SSE messages with 15-second
// keepalive comments to maintain the connection.
func (h *LiveHandler) HandleSSE(c *gin.Context) {
	token := c.Query("token")
	if !h.validateToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "unauthorized"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	eventCh, unsubscribe := GlobalEventHub.Subscribe()
	defer unsubscribe()

	ctx := c.Request.Context()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-eventCh:
			data, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("Failed to marshal recognition event", zap.Error(err))
				continue
			}
			_, _ = fmt.Fprintf(c.Writer, "event: recognition\ndata: %s\n\n", data)
			c.Writer.Flush()
		case <-ticker.C:
			_, _ = fmt.Fprintf(c.Writer, ": keepalive\n\n")
			c.Writer.Flush()
		case <-ctx.Done():
			h.logger.Info("SSE client disconnected")
			return
		}
	}
}
