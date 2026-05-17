package camera

import (
	"context"
	"log"
	"sync"

	"github.com/sims/campusvision/stream-gateway/internal/config"
	"github.com/sims/campusvision/stream-gateway/internal/kafka"
)

// context.Background() for streams added dynamically (not tied to main ctx lifecycle;
// individual streams manage their own cancellation via Stop/stopCh).
var backgroundCtx = context.Background()

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

// AddCamera starts a new camera stream. No-op if camera already exists or is disabled.
func (m *Manager) AddCamera(cfg config.CameraConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.streams[cfg.ID]; exists {
		log.Printf("[manager] camera %s already exists, skipping", cfg.ID)
		return
	}
	if !cfg.Enabled {
		return
	}

	stream := NewStream(cfg, m.cfg, m.rtspCfg, m.producer,
		func(id string, s CameraStatus) { m.UpdateStatus(id, s) },
	)
	m.streams[cfg.ID] = stream
	m.statuses[cfg.ID] = CameraStatus{
		CameraID:  cfg.ID,
		Building:  cfg.Building,
		Connected: false,
	}
	go stream.Run(backgroundCtx)
	log.Printf("[manager] camera stream added: %s (%s栋)", cfg.ID, cfg.Building)
}

// RemoveCamera stops and removes a camera stream by ID.
func (m *Manager) RemoveCamera(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stream, ok := m.streams[id]; ok {
		stream.Stop()
		delete(m.streams, id)
		delete(m.statuses, id)
		log.Printf("[manager] camera stream removed: %s", id)
	}
}

// DiffAndSync diffs current streams against desired camera configs and reconciles them.
// Adds new cameras, removes stale ones, skips unchanged.
func (m *Manager) DiffAndSync(desired []config.CameraConfig) {
	m.mu.RLock()
	currentIDs := make(map[string]bool, len(m.streams))
	for id := range m.streams {
		currentIDs[id] = true
	}
	m.mu.RUnlock()

	desiredIDs := make(map[string]config.CameraConfig, len(desired))
	for _, cam := range desired {
		desiredIDs[cam.ID] = cam
	}

	for id := range currentIDs {
		if _, ok := desiredIDs[id]; !ok {
			go m.RemoveCamera(id)
		}
	}

	for id, cfg := range desiredIDs {
		m.mu.RLock()
		_, exists := m.streams[id]
		m.mu.RUnlock()

		if !exists && cfg.Enabled {
			m.AddCamera(cfg)
		} else if exists && !cfg.Enabled {
			go m.RemoveCamera(id)
		}
	}
}
