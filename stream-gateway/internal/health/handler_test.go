package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sims/campusvision/stream-gateway/internal/camera"
	"github.com/sims/campusvision/stream-gateway/internal/config"
)

func makeManager() *camera.Manager {
	cfg := config.FrameConfig{FPSDay: 5, FPSNight: 1, JPEGQuality: 85, Width: 1280, Height: 720}
	rtspCfg := config.RTSPConfig{ReconnectInterval: 5, ReadTimeout: 10, MaxReconnectAttempts: 3}
	return camera.NewManager(cfg, rtspCfg, nil)
}

func TestCameraHealth_ExistingCamera(t *testing.T) {
	manager := makeManager()
	manager.AddCamera(config.CameraConfig{ID: "cam-a", Building: "A", RTSPURL: "rtsp://test", Enabled: true})

	handler := NewHandler(manager)
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cameras/cam-a/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["camera_id"] != "cam-a" {
		t.Errorf("expected camera_id=cam-a, got %v", result["camera_id"])
	}
}

func TestCameraHealth_UnknownCamera(t *testing.T) {
	manager := makeManager()

	handler := NewHandler(manager)
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cameras/nonexistent/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["error"] != "camera not found" {
		t.Errorf("expected error=camera not found, got %v", result["error"])
	}
}
