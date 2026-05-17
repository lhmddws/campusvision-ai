package camera

import (
	"context"
	"log"
	"sync"

	"github.com/sims/campusvision/stream-gateway/internal/config"
	"github.com/sims/campusvision/stream-gateway/internal/kafka"
)

type CameraStatus struct {
	CameraID     string `json:"camera_id"`
	Building     string `json:"building"`
	Connected    bool   `json:"connected"`
	FPS          float64 `json:"fps"`
	LastFrameTime string `json:"last_frame_time"`
	FramesSent   int64  `json:"frames_sent"`
	UptimeSeconds int64 `json:"uptime_seconds"`
}

type Manager struct {
	mu       sync.RWMutex
	streams  map[string]*Stream
	statuses map[string]CameraStatus
	producer *kafka.Producer
	cfg      config.FrameConfig
	rtspCfg  config.RTSPConfig
}

func NewManager(cfg config.FrameConfig, rtspCfg config.RTSPConfig, producer *kafka.Producer) *Manager {
	return &Manager{
		streams:  make(map[string]*Stream),
		statuses: make(map[string]CameraStatus),
		producer: producer,
		cfg:      cfg,
		rtspCfg:  rtspCfg,
	}
}

func (m *Manager) Start(ctx context.Context, cameras []config.CameraConfig) {
	for _, cam := range cameras {
		if !cam.Enabled {
			continue
		}
		stream := NewStream(cam, m.cfg, m.rtspCfg, m.producer,
			func(id string, s CameraStatus) { m.UpdateStatus(id, s) },
		)
		m.mu.Lock()
		m.streams[cam.ID] = stream
		m.statuses[cam.ID] = CameraStatus{
			CameraID:  cam.ID,
			Building:  cam.Building,
			Connected: false,
		}
		m.mu.Unlock()

		go stream.Run(ctx)
		log.Printf("Camera stream started: %s (%s栋)", cam.ID, cam.Building)
	}
}

func (m *Manager) Stop() {
	for id, stream := range m.streams {
		stream.Stop()
		log.Printf("Camera stream stopped: %s", id)
	}
}

func (m *Manager) Statuses() map[string]CameraStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]CameraStatus, len(m.statuses))
	for k, v := range m.statuses {
		result[k] = v
	}
	return result
}

func (m *Manager) UpdateStatus(cameraID string, status CameraStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[cameraID] = status
}
