package state

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// ── Types ──────────────────────────────────────────────────────────────────────

type CameraDef struct {
	Building string `json:"building"`
	Label    string `json:"label"`
	Color    string `json:"color"`
}

type Config struct {
	JPEGQuality          int       `json:"jpeg_quality"`
	FrameWidth           int       `json:"frame_width"`
	FrameHeight          int       `json:"frame_height"`
	FPS                  int       `json:"fps"`
	ConfidenceThreshold  float64   `json:"confidence_threshold"`
	MinFaceSize          int       `json:"min_face_size"`
	MatchThreshold       float64   `json:"match_threshold"`
	CacheTTL             int       `json:"cache_ttl"`
	RoiLineX             float64   `json:"roi_line_x"`
	MinTrackPoints       int       `json:"min_track_points"`
	DedupWindowSeconds   int       `json:"dedup_window_seconds"`
	StrangerAlertEnabled bool      `json:"stranger_alert_enabled"`
	StrangerAlertThresh  float64   `json:"stranger_alert_threshold"`
	NightModeEnabled     bool      `json:"night_mode_enabled"`
	NightModeStartHour   int       `json:"night_mode_start_hour"`
	NightModeEndHour     int       `json:"night_mode_end_hour"`
	MotionThreshold      float64   `json:"motion_threshold"`
	DynamicExtraction    bool      `json:"dynamic_extraction"`
	CameraSource         string    `json:"camera_source"`
	WebcamDevice         int       `json:"webcam_device"`
	UseFakeData          bool      `json:"use_fake_data"`
	TestPeople           []string  `json:"test_people"`
	DBDsn                string    `json:"db_dsn"`
}

type EventEntry struct {
	Time      string `json:"time"`
	CameraID  string `json:"camera_id"`
	Building  string `json:"building"`
	EventType string `json:"event_type"`
	Detail    string `json:"detail"`
}

type FaceEntry struct {
	Name       string `json:"name"`
	StudentID  string `json:"student_id"`
	EnrolledAt string `json:"enrolled_at"`
	ImageURL   string `json:"image_url"`
}

type RecognitionResult struct {
	CameraID   string  `json:"camera_id"`
	EventType  string  `json:"event_type"`
	StudentID  string  `json:"student_id"`
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	IsStranger bool    `json:"is_stranger"`
	Timestamp  int64   `json:"timestamp"`
}

// ── Defaults ───────────────────────────────────────────────────────────────────

var DefaultCameras = map[string]CameraDef{
	"cam-a": {Building: "A", Label: "A栋入口", Color: "#2980b9"},
	"cam-b": {Building: "B", Label: "B栋入口", Color: "#27ae60"},
	"cam-c": {Building: "C", Label: "C栋入口", Color: "#8e44ad"},
	"cam-d": {Building: "D", Label: "D栋入口", Color: "#e67e22"},
}

var DefaultConfig = Config{
	JPEGQuality:          80,
	FrameWidth:           640,
	FrameHeight:          360,
	FPS:                  5,
	ConfidenceThreshold:  0.6,
	MinFaceSize:          80,
	MatchThreshold:       0.65,
	CacheTTL:             3600,
	RoiLineX:             0.5,
	MinTrackPoints:       3,
	DedupWindowSeconds:   10,
	StrangerAlertEnabled: true,
	StrangerAlertThresh:  0.45,
	NightModeEnabled:     true,
	NightModeStartHour:   22,
	NightModeEndHour:     6,
	MotionThreshold:      0.05,
	DynamicExtraction:    true,
	CameraSource:         "simulated",
	WebcamDevice:         0,
	UseFakeData:          true,
	TestPeople: []string{
		"张三 (2024001)", "李四 (2024002)", "王五 (2024003)",
		"赵六 (2024004)", "孙七 (2024005)", "周八 (2024006)",
	},
}

const EventLogMax = 300

// ── State ──────────────────────────────────────────────────────────────────────

type State struct {
	mu            sync.RWMutex
	Cameras       map[string]CameraDef
	Config        Config
	events        []EventEntry // ring buffer
	LatestFrames  map[string][]byte
	WebcamRunning map[string]bool
	WebcamIndex   map[string]*int
	FacesMeta     []FaceEntry
	RecognitionResults map[string][]RecognitionResult
	SSEClients    map[chan string]bool
	sseMu         sync.RWMutex
	StartTime     time.Time
	EventCounter  int64
	KafkaProducer *kafka.Writer
	KafkaEventPub *kafka.Writer
	MariaDB       *sql.DB
}

func New() *State {
	cfg := DefaultConfig
	people := make([]string, len(cfg.TestPeople))
	copy(people, cfg.TestPeople)
	cfg.TestPeople = people

	cameras := make(map[string]CameraDef)
	for k, v := range DefaultCameras {
		cameras[k] = v
	}

	return &State{
		Cameras:       cameras,
		Config:        cfg,
		events:        make([]EventEntry, 0, EventLogMax),
		LatestFrames:  make(map[string][]byte),
		WebcamRunning: make(map[string]bool),
		WebcamIndex:   make(map[string]*int),
		FacesMeta:          make([]FaceEntry, 0),
		RecognitionResults: make(map[string][]RecognitionResult),
		SSEClients:    make(map[chan string]bool),
		StartTime:     time.Now(),
	}
}

// ── Thread-safe accessors ──────────────────────────────────────────────────────

func (s *State) Lock()   { s.mu.Lock() }
func (s *State) Unlock() { s.mu.Unlock() }
func (s *State) RLock()   { s.mu.RLock() }
func (s *State) RUnlock() { s.mu.RUnlock() }

func (s *State) GetCameras() map[string]CameraDef {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := make(map[string]CameraDef, len(s.Cameras))
	for k, v := range s.Cameras {
		m[k] = v
	}
	return m
}

func (s *State) GetCamera(id string) (CameraDef, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.Cameras[id]
	return c, ok
}

func (s *State) GetConfig() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Config
}

func (s *State) GetLatestFrame(id string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.LatestFrames[id]
	return f, ok
}

func (s *State) SetLatestFrame(id string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LatestFrames[id] = data
}

func (s *State) StoreRecognitionResult(cameraID string, r RecognitionResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RecognitionResults[cameraID] = append(s.RecognitionResults[cameraID], r)
	if len(s.RecognitionResults[cameraID]) > 50 {
		s.RecognitionResults[cameraID] = s.RecognitionResults[cameraID][1:]
	}
}

func (s *State) GetRecognitionResults(cameraID string) []RecognitionResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RecognitionResults[cameraID]
}

// ── SSE Client Management ─────────────────────────────────────────────────────

func (s *State) AddSSEClient(ch chan string) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	s.SSEClients[ch] = true
}

func (s *State) RemoveSSEClient(ch chan string) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	delete(s.SSEClients, ch)
}

func (s *State) BroadcastSSE(event string) {
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()
	for ch := range s.SSEClients {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *State) LogEvent(cameraID, eventType, detail string) EventEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	building := ""
	if c, ok := s.Cameras[cameraID]; ok {
		building = c.Building
	} else {
		building = cameraID
	}

	entry := EventEntry{
		Time:      time.Now().Format("15:04:05"),
		CameraID:  cameraID,
		Building:  building,
		EventType: eventType,
		Detail:    detail,
	}

	s.events = append([]EventEntry{entry}, s.events...)
	if len(s.events) > EventLogMax {
		s.events = s.events[:EventLogMax]
	}
	return entry
}

func (s *State) GetEvents(limit int) []EventEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	result := make([]EventEntry, limit)
	copy(result, s.events[:limit])
	return result
}

func (s *State) ClearEvents() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
}

func (s *State) EventCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

func (s *State) PickRandomPerson() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.Config.TestPeople) == 0 {
		return ""
	}
	return s.Config.TestPeople[rand.Intn(len(s.Config.TestPeople))]
}

// WebcamIndex returns a copy of the webcam index map
func (s *State) WebcamIndexCopy() map[string]*int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := make(map[string]*int, len(s.WebcamIndex))
	for k, v := range s.WebcamIndex {
		if v != nil {
			val := *v
			m[k] = &val
		} else {
			m[k] = nil
		}
	}
	return m
}

func (s *State) IsWebcamRunning(id string) bool {
	return s.WebcamRunning[id]
}

// Get next event sequence number
func (s *State) NextSeq() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EventCounter++
	return s.EventCounter
}

// Uptime returns seconds since start
func (s *State) Uptime() int {
	return int(time.Since(s.StartTime).Seconds())
}

// GetFaceMetadataPath returns the absolute path for face metadata
func GetFaceMetadataPath(baseDir string) string {
	return baseDir + "/face_data/metadata.json"
}

// GetFaceImageDir returns the absolute path for face images
func GetFaceImageDir(baseDir string) string {
	return baseDir + "/face_data/images"
}

// GetFaceImagePath returns path for a specific face image
func GetFaceImagePath(baseDir, name string) string {
	return fmt.Sprintf("%s/face_data/images/%s.jpg", baseDir, name)
}
