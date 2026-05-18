package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sims/campusvision/stream-gateway/internal/camera"
	"github.com/sims/campusvision/stream-gateway/internal/config"
)

func newTestManager() *camera.Manager {
	return camera.NewManager(
		config.FrameConfig{},
		config.RTSPConfig{},
		nil, // producer is nil; AddCamera with enabled=false avoids stream startup
	)
}

func setupHandler(t *testing.T, mgr *camera.Manager, key string) *Handler {
	t.Helper()
	return NewHandler(mgr, Config{
		Port:          8081,
		BindAddress:   "127.0.0.1",
		ManagementKey: key,
	})
}

func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestListCameras(t *testing.T) {
	mgr := newTestManager()
	// Pre-populate via UpdateStatus (avoids needing a real stream)
	mgr.UpdateStatus("CAM_A", camera.CameraStatus{
		CameraID: "CAM_A",
		Building: "A",
	})
	mgr.UpdateStatus("CAM_B", camera.CameraStatus{
		CameraID: "CAM_B",
		Building: "B",
	})

	h := setupHandler(t, mgr, "")
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cameras", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var items []camera.CameraStatus
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 cameras, got %d", len(items))
	}
	if items[0].CameraID != "CAM_A" && items[1].CameraID != "CAM_A" {
		t.Fatal("expected CAM_A in response")
	}
}

func TestAddCamera(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "")
	mux := http.NewServeMux()
	h.Register(mux)

	body := mustMarshal(t, map[string]interface{}{
		"id":       "CAM_NEW",
		"building": "X",
		"rtsp_url": "rtsp://example.com/stream",
		"enabled":  false, // false avoids starting a real stream
	})

	req := httptest.NewRequest(http.MethodPost, "/cameras", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "added" || resp["id"] != "CAM_NEW" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAddCameraMissingFields(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "")
	mux := http.NewServeMux()
	h.Register(mux)

	// Missing building
	body := mustMarshal(t, map[string]string{"id": "CAM_X"})
	req := httptest.NewRequest(http.MethodPost, "/cameras", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDeleteCamera(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "")
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodDelete, "/cameras/CAM_X", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestGetCameraByID(t *testing.T) {
	mgr := newTestManager()
	mgr.UpdateStatus("CAM_A", camera.CameraStatus{
		CameraID: "CAM_A",
		Building: "A",
	})

	h := setupHandler(t, mgr, "")
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cameras/CAM_A", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var status camera.CameraStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if status.CameraID != "CAM_A" || status.Building != "A" {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestGetUnknownCamera(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "")
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cameras/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "")
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPut, "/cameras", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsWithoutKey(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "secret-key")
	mux := http.NewServeMux()
	h.Register(mux)

	// Request without the management key header
	req := httptest.NewRequest(http.MethodGet, "/cameras", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareAllowsWithValidKey(t *testing.T) {
	mgr := newTestManager()
	mgr.UpdateStatus("CAM_A", camera.CameraStatus{
		CameraID: "CAM_A",
		Building: "A",
	})

	h := setupHandler(t, mgr, "secret-key")
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cameras", nil)
	req.Header.Set("X-Management-Key", "secret-key")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsWithWrongKey(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "secret-key")
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cameras", nil)
	req.Header.Set("X-Management-Key", "wrong-key")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestEmptyKeySkipsAuth(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "") // empty key → auth disabled
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/cameras", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestInvalidJSON(t *testing.T) {
	mgr := newTestManager()
	h := setupHandler(t, mgr, "")
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/cameras", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
