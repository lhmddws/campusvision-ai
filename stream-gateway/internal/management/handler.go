package management

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sims/campusvision/stream-gateway/internal/camera"
	"github.com/sims/campusvision/stream-gateway/internal/config"
)

// Config holds the management HTTP API configuration.
type Config struct {
	Port          int    `yaml:"port" json:"port"`
	BindAddress   string `yaml:"bind_address" json:"bind_address"`
	ManagementKey string `yaml:"management_key" json:"management_key"`
}

// Handler exposes camera lifecycle management over HTTP.
type Handler struct {
	manager *camera.Manager
	cfg     Config
}

// NewHandler creates a new management Handler.
func NewHandler(manager *camera.Manager, cfg Config) *Handler {
	return &Handler{manager: manager, cfg: cfg}
}

func (h *Handler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.cfg.ManagementKey != "" {
			key := r.Header.Get("X-Management-Key")
			if key == "" || subtle.ConstantTimeCompare([]byte(key), []byte(h.cfg.ManagementKey)) != 1 {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

// Register registers management API routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/cameras", h.auth(h.handleCameras))
	mux.HandleFunc("/cameras/", h.auth(h.handleCameraByID))
}

func (h *Handler) handleCameras(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		statuses := h.manager.Statuses()
		items := make([]camera.CameraStatus, 0, len(statuses))
		for _, s := range statuses {
			items = append(items, s)
		}
		writeJSON(w, http.StatusOK, items)

	case http.MethodPost:
		var body struct {
			ID       string `json:"id"`
			Building string `json:"building"`
			RTSPURL  string `json:"rtsp_url"`
			Enabled  bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if body.ID == "" || body.Building == "" {
			http.Error(w, `{"error":"id and building are required"}`, http.StatusBadRequest)
			return
		}

		camCfg := config.CameraConfig{
			ID:       body.ID,
			Building: body.Building,
			Type:     "RTSP",
			RTSPURL:  body.RTSPURL,
			Enabled:  body.Enabled,
		}
		h.manager.AddCamera(camCfg)
		writeJSON(w, http.StatusCreated, map[string]string{"status": "added", "id": body.ID})

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCameraByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/cameras/")
	parts := strings.SplitN(path, "/", 2)
	cameraID := parts[0]

	if cameraID == "" {
		http.Error(w, `{"error":"camera ID required"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		statuses := h.manager.Statuses()
		status, ok := statuses[cameraID]
		if !ok {
			http.Error(w, `{"error":"camera not found"}`, http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, status)

	case http.MethodDelete:
		h.manager.RemoveCamera(cameraID)
		writeJSON(w, http.StatusNoContent, nil)

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
