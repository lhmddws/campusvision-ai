package health

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sims/campusvision/stream-gateway/internal/camera"
)

type CameraStatusItem struct {
	CameraID      string  `json:"camera_id"`
	Building      string  `json:"building"`
	Connected     bool    `json:"connected"`
	FPS           float64 `json:"fps"`
	LastFrameTime string  `json:"last_frame_time"`
	FramesSent    int64   `json:"frames_sent"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

type Handler struct {
	manager   *camera.Manager
	startedAt time.Time
}

func NewHandler(manager *camera.Manager) *Handler {
	return &Handler{manager: manager, startedAt: time.Now()}
}

func (h *Handler) HandleCameraHealth(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/cameras/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] != "health" {
		http.Error(w, `{"error":"invalid path"}`, http.StatusBadRequest)
		return
	}
	cameraID := parts[0]
	if cameraID == "" {
		http.Error(w, `{"error":"camera ID required"}`, http.StatusBadRequest)
		return
	}

	statuses := h.manager.Statuses()
	status, ok := statuses[cameraID]
	if !ok {
		http.Error(w, `{"error":"camera not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("[health] json encode error: %v", err)
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		statuses := h.manager.Statuses()

		items := make([]CameraStatusItem, 0, len(statuses))
		for _, s := range statuses {
			uptime := int64(0)
			if s.Connected {
				uptime = int64(time.Since(h.startedAt).Seconds())
			}
			items = append(items, CameraStatusItem{
				CameraID:      s.CameraID,
				Building:      s.Building,
				Connected:     s.Connected,
				FPS:           s.FPS,
				LastFrameTime: s.LastFrameTime,
				FramesSent:    s.FramesSent,
				UptimeSeconds: uptime,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "UP",
			"cameras": items,
		}); err != nil {
			log.Printf("[health] json encode error: %v", err)
		}
	})

	mux.HandleFunc("/cameras/", h.HandleCameraHealth)
}
