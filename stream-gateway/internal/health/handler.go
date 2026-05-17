package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sims/campusvision/stream-gateway/internal/camera"
)

type HealthResponse struct {
	Status  string                `json:"status"`
	Cameras []camera.CameraStatus `json:"cameras"`
}

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
	manager *camera.Manager
}

func NewHandler(manager *camera.Manager) *Handler {
	return &Handler{manager: manager}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	statuses := h.manager.Statuses()
	items := make([]CameraStatusItem, 0, len(statuses))

	for _, s := range statuses {
		items = append(items, CameraStatusItem{
			CameraID:      s.CameraID,
			Building:      s.Building,
			Connected:     s.Connected,
			LastFrameTime: s.LastFrameTime,
			FramesSent:    s.FramesSent,
			UptimeSeconds: s.UptimeSeconds,
		})
	}

	resp := HealthResponse{
		Status:  "UP",
		Cameras: make([]camera.CameraStatus, 0),
	}

	for _, s := range statuses {
		resp.Cameras = append(resp.Cameras, s)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		statuses := h.manager.Statuses()
		now := time.Now()

		items := make([]CameraStatusItem, 0, len(statuses))
		for _, s := range statuses {
			uptime := int64(0)
			if s.Connected {
				uptime = int64(now.Sub(time.UnixMilli(0)).Seconds())
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "UP",
			"cameras": items,
		})
	})
}
