package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// PushClient sends camera lifecycle notifications to the stream-gateway.
type PushClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewPushClient creates a PushClient with the given base URL and API key.
func NewPushClient(baseURL, apiKey string) *PushClient {
	return &PushClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *PushClient) doPost(path string, body interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("X-Management-Key", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

// NotifyRegister notifies the stream-gateway about a newly registered camera.
func (c *PushClient) NotifyRegister(camera entity.DormCamera) error {
	return c.doPost("/api/cameras/register", map[string]interface{}{
		"camera_id": camera.CameraID,
		"building":  camera.Building,
		"name":      camera.Name,
		"rtsp_url":  camera.RtspURL,
		"status":    camera.Status,
		"enabled":   camera.Enabled,
	})
}

// NotifyUpdate notifies the stream-gateway about a camera configuration change.
func (c *PushClient) NotifyUpdate(cameraID string, camera entity.DormCamera) error {
	return c.doPost("/api/cameras/update", map[string]interface{}{
		"camera_id": cameraID,
		"building":  camera.Building,
		"name":      camera.Name,
		"rtsp_url":  camera.RtspURL,
		"status":    camera.Status,
		"enabled":   camera.Enabled,
	})
}

// NotifyDelete notifies the stream-gateway that a camera has been removed.
func (c *PushClient) NotifyDelete(cameraID string) error {
	return c.doPost("/api/cameras/delete", map[string]string{
		"camera_id": cameraID,
	})
}
