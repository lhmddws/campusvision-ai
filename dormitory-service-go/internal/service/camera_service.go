package service

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/client"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"github.com/sims/campusvision/dormitory-service-go/internal/util"
	"go.uber.org/zap"
)

// CameraService handles camera lifecycle management.
type CameraService struct {
	cameraRepo    *repository.CameraRepository
	eventLogRepo  *repository.EventLogRepository
	cameraLogRepo *repository.CameraLogRepository
	pushClient    *client.PushClient
	gatewayURL    string
	maxCameras    int
	logger        *zap.Logger
	mu            sync.Mutex
}

// NewCameraService creates a new CameraService.
func NewCameraService(
	cameraRepo *repository.CameraRepository,
	eventLogRepo *repository.EventLogRepository,
	cameraLogRepo *repository.CameraLogRepository,
	pushClient *client.PushClient,
	gatewayURL string,
	maxCameras int,
	logger *zap.Logger,
) *CameraService {
	if maxCameras <= 0 {
		maxCameras = 50
	}
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	return &CameraService{
		cameraRepo:    cameraRepo,
		eventLogRepo:  eventLogRepo,
		cameraLogRepo: cameraLogRepo,
		pushClient:    pushClient,
		gatewayURL:    gatewayURL,
		maxCameras:    maxCameras,
		logger:        logger,
	}
}

func (s *CameraService) log() *zap.Logger {
	if s.logger == nil {
		return zap.NewNop()
	}
	return s.logger
}

// RegisterCamera creates a new camera entry with RTSP URL parsing and push notification.
func (s *CameraService) RegisterCamera(dto dto.CameraCreateDTO) (*entity.DormCamera, error) {
	s.mu.Lock()
	all, err := s.cameraRepo.FindAll(serviceCtx())
	if err != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("count cameras: %w", err)
	}
	if len(all) >= s.maxCameras {
		s.mu.Unlock()
		return nil, ErrCameraLimitExceeded
	}
	s.mu.Unlock()

	var passwordEnc, nonce jsontype.NullString
	if dto.RtspURL != "" {
		if parsed, err := url.Parse(dto.RtspURL); err == nil {
			if dto.Host == "" && parsed.Host != "" {
				if h, _, _ := net.SplitHostPort(parsed.Host); h != "" {
					dto.Host = h
				} else {
					dto.Host = parsed.Host
				}
			}
			if dto.Port == 0 {
				if _, p, err := net.SplitHostPort(parsed.Host); err == nil {
					if port, atoiErr := strconv.Atoi(p); atoiErr == nil {
						dto.Port = port
					}
				}
			}
			if dto.Username == "" && parsed.User != nil {
				dto.Username = parsed.User.Username()
			}
			if dto.Path == "" && parsed.Path != "" {
				dto.Path = parsed.Path
			}
			if parsed.User != nil {
				pass, hasPass := parsed.User.Password()
				if hasPass && pass != "" {
					if ep, encErr := util.EncryptPassword(pass); encErr == nil {
						passwordEnc = toNullString(ep.Ciphertext)
						nonce = toNullString(ep.Nonce)
						s.log().Info("password encrypted", zap.String("camera_id", dto.CameraID))
					}
				}
			}
		} else {
			s.log().Warn("failed to parse RTSP URL", zap.Error(err))
		}
	}

	cam := &entity.DormCamera{
		CameraID:    dto.CameraID,
		Building:    dto.Building,
		Name:        dto.Name,
		RtspURL:     dto.RtspURL,
		Direction:   dto.Direction,
		Resolution:  dto.Resolution,
		Status:      "unknown",
		Enabled:     true,
		Remark:      toNullString(dto.Remark),
		PasswordEnc: passwordEnc,
		Nonce:       nonce,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	id, err := s.cameraRepo.Create(serviceCtx(), cam)
	if err != nil {
		return nil, fmt.Errorf("create camera: %w", err)
	}
	cam.ID = id

	s.insertCameraLog(cam, "", "unknown", "Camera registered")

	// Push notification (best-effort)
	if s.pushClient != nil {
		if err := s.pushClient.NotifyRegister(*cam); err != nil {
			s.log().Warn("push notification failed for register", zap.Error(err))
		}
	}

	return cam, nil
}

// UpdateCamera patches an existing camera and sends push notification.
func (s *CameraService) UpdateCamera(cameraID string, dto dto.CameraUpdateDTO) (*entity.DormCamera, error) {
	cam, err := s.cameraRepo.FindByCameraID(serviceCtx(), cameraID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find camera: %w", err)
	}

	if dto.Name != "" {
		cam.Name = dto.Name
	}
	if dto.RtspURL != "" {
		cam.RtspURL = dto.RtspURL
	}
	if dto.Building != "" {
		cam.Building = dto.Building
	}
	if dto.Direction != "" {
		cam.Direction = dto.Direction
	}
	if dto.Resolution != "" {
		cam.Resolution = dto.Resolution
	}
	if dto.Status != "" {
		cam.Status = dto.Status
	}
	if dto.Enabled != nil {
		cam.Enabled = *dto.Enabled
	}
	if dto.Remark != "" {
		cam.Remark = toNullString(dto.Remark)
	}
	cam.UpdatedAt = time.Now()

	if err := s.cameraRepo.Update(serviceCtx(), cam); err != nil {
		return nil, fmt.Errorf("update camera: %w", err)
	}

	if s.pushClient != nil {
		if err := s.pushClient.NotifyUpdate(cameraID, *cam); err != nil {
			s.log().Warn("push notification failed for update", zap.Error(err))
		}
	}

	return cam, nil
}

// DeleteCamera removes a camera and sends push notification.
func (s *CameraService) DeleteCamera(cameraID string) error {
	cam, err := s.cameraRepo.FindByCameraID(serviceCtx(), cameraID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("find camera: %w", err)
	}

	s.insertCameraLog(cam, cam.Status, "DELETED", "Camera deleted")

	if err := s.cameraRepo.Delete(serviceCtx(), cam.ID); err != nil {
		return fmt.Errorf("delete camera: %w", err)
	}

	if s.pushClient != nil {
		if err := s.pushClient.NotifyDelete(cameraID); err != nil {
			s.log().Warn("push notification failed for delete", zap.Error(err))
		}
	}

	return nil
}

// GetByCameraID finds a camera by its unique camera ID.
// Returns ErrNotFound if the camera does not exist.
func (s *CameraService) GetByCameraID(cameraID string) (*entity.DormCamera, error) {
	cam, err := s.cameraRepo.FindByCameraID(serviceCtx(), cameraID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find camera: %w", err)
	}
	return cam, nil
}

// GetCameras lists cameras, optionally filtered by building.
func (s *CameraService) GetCameras(building string) ([]entity.DormCamera, error) {
	if building != "" {
		return s.cameraRepo.FindByBuilding(serviceCtx(), building)
	}
	return s.cameraRepo.FindAll(serviceCtx(), "camera_id ASC")
}

// GetCameraStatus returns a status summary for all cameras (optionally filtered by building).
func (s *CameraService) GetCameraStatus(building string) (map[string]interface{}, error) {
	var cameras []entity.DormCamera
	var err error
	if building != "" {
		cameras, err = s.cameraRepo.FindByBuilding(serviceCtx(), building)
	} else {
		cameras, err = s.cameraRepo.FindAll(serviceCtx())
	}
	if err != nil {
		return nil, fmt.Errorf("list cameras: %w", err)
	}

	type statusItem struct {
		Building        string      `json:"building"`
		CameraID        string      `json:"camera_id"`
		Status          string      `json:"status"`
		LastHealthCheck interface{} `json:"last_health_check"`
	}

	items := make([]statusItem, 0, len(cameras))
	var total, online, offline, idle int
	for _, c := range cameras {
		total++
		switch strings.ToLower(c.Status) {
		case "online":
			online++
		case "offline":
			offline++
		case "idle":
			idle++
		}
		var lastCheck interface{}
		if c.LastHealthCheck.Valid {
			lastCheck = c.LastHealthCheck.Time
		}
		items = append(items, statusItem{
			Building:        c.Building,
			CameraID:        c.CameraID,
			Status:          c.Status,
			LastHealthCheck: lastCheck,
		})
	}

	return map[string]interface{}{
		"cameras": items,
		"summary": map[string]int{
			"total":   total,
			"online":  online,
			"offline": offline,
			"idle":    idle,
		},
	}, nil
}

// UpdateLastEventTime updates the last_event_time for a camera to now.
func (s *CameraService) UpdateLastEventTime(cameraID string) error {
	return s.cameraRepo.UpdateLastEventTime(serviceCtx(), cameraID)
}

// HealthCheck pings the stream-gateway health endpoint and updates camera status.
func (s *CameraService) HealthCheck(cameraID string) error {
	cam, err := s.cameraRepo.FindByCameraID(serviceCtx(), cameraID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("find camera: %w", err)
	}

	oldStatus := cam.Status
	newStatus := "offline"
	gatewayURL := s.gatewayURL + "/health"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(gatewayURL)
	if err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			newStatus = "online"
		}
	}

	if err := s.cameraRepo.UpdateStatus(serviceCtx(), cameraID, newStatus, 0, 0); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if err := s.cameraRepo.UpdateHealthCheck(serviceCtx(), cameraID); err != nil {
		return fmt.Errorf("update health check: %w", err)
	}

	if oldStatus != newStatus {
		reason := "Health check"
		if newStatus == "offline" {
			reason = "Health check failed"
		}
		s.insertCameraLog(cam, oldStatus, newStatus, reason)
	}

	return nil
}

// ListEnabledCameras returns all cameras where enabled = true.
func (s *CameraService) ListEnabledCameras() ([]entity.DormCamera, error) {
	return s.cameraRepo.FindEnabled(serviceCtx())
}

// ListOnlineCameras returns all cameras with status = "online" and enabled = true.
func (s *CameraService) ListOnlineCameras() ([]entity.DormCamera, error) {
	all, err := s.cameraRepo.FindAll(serviceCtx())
	if err != nil {
		return nil, err
	}
	online := make([]entity.DormCamera, 0, len(all))
	for _, c := range all {
		if strings.EqualFold(c.Status, "online") && c.Enabled {
			online = append(online, c)
		}
	}
	return online, nil
}

// QuerySnapshots returns paginated event logs for a camera within a time range.
func (s *CameraService) QuerySnapshots(cameraID string, startTime, endTime time.Time, page, size int) ([]entity.DormEventLog, int64, error) {
	return s.eventLogRepo.FindWithPagination(serviceCtx(), "", cameraID, "", "", &startTime, &endTime, page, size)
}

func (s *CameraService) insertCameraLog(cam *entity.DormCamera, statusFrom, statusTo, reason string) {
	entry := &entity.DormCameraLog{
		CameraID: cam.CameraID,
		Building: cam.Building,
		StatusTo: statusTo,
		Reason:   toNullString(reason),
		CreatedAt: time.Now(),
	}
	if statusFrom != "" {
		entry.StatusFrom = toNullString(statusFrom)
	}
	if _, err := s.cameraLogRepo.Create(serviceCtx(), entry); err != nil {
		s.log().Warn("failed to insert camera log", zap.Error(err))
	}
}

// toNullString converts a plain string to jsontype.NullString.
func toNullString(s string) jsontype.NullString {
	return jsontype.NewNullString(s)
}
