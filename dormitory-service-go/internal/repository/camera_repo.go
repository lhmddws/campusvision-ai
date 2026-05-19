package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// CameraRepository handles dorm_camera table operations.
type CameraRepository struct {
	*BaseRepository[entity.DormCamera]
}

// NewCameraRepository creates a new CameraRepository.
func NewCameraRepository(db *sqlx.DB) *CameraRepository {
	return &CameraRepository{
		BaseRepository: NewBaseRepository[entity.DormCamera](db, "dorm_camera"),
	}
}

// FindByCameraID finds a camera by its unique camera_id.
func (r *CameraRepository) FindByCameraID(cameraID string) (*entity.DormCamera, error) {
	var cam entity.DormCamera
	query := "SELECT * FROM dorm_camera WHERE camera_id = ? LIMIT 1"
	err := r.DB.Get(&cam, query, cameraID)
	if err != nil {
		return nil, fmt.Errorf("find camera by id %s: %w", cameraID, err)
	}
	return &cam, nil
}

// FindByBuilding finds all cameras in a given building.
func (r *CameraRepository) FindByBuilding(building string) ([]entity.DormCamera, error) {
	var cams []entity.DormCamera
	query := "SELECT * FROM dorm_camera WHERE building = ? ORDER BY camera_id"
	err := r.DB.Select(&cams, query, building)
	if err != nil {
		return nil, fmt.Errorf("find cameras by building %s: %w", building, err)
	}
	return cams, nil
}

// FindByStatus finds all cameras with a given status.
func (r *CameraRepository) FindByStatus(status string) ([]entity.DormCamera, error) {
	var cams []entity.DormCamera
	query := "SELECT * FROM dorm_camera WHERE status = ? ORDER BY camera_id"
	err := r.DB.Select(&cams, query, status)
	if err != nil {
		return nil, fmt.Errorf("find cameras by status %s: %w", status, err)
	}
	return cams, nil
}

// FindEnabled finds all enabled cameras.
func (r *CameraRepository) FindEnabled() ([]entity.DormCamera, error) {
	var cams []entity.DormCamera
	query := "SELECT * FROM dorm_camera WHERE enabled = 1 ORDER BY camera_id"
	err := r.DB.Select(&cams, query)
	if err != nil {
		return nil, fmt.Errorf("find enabled cameras: %w", err)
	}
	return cams, nil
}

// UpdateStatus updates a camera's status and related fields.
func (r *CameraRepository) UpdateStatus(cameraID string, status string, fps float64, totalFrames int64) error {
	query := `UPDATE dorm_camera SET status = ?, fps_current = ?, total_frames = ?, last_heartbeat = NOW() WHERE camera_id = ?`
	_, err := r.DB.Exec(query, status, fps, totalFrames, cameraID)
	if err != nil {
		return fmt.Errorf("update camera status: %w", err)
	}
	return nil
}

// UpdateLastEventTime updates the last_event_time for a camera.
func (r *CameraRepository) UpdateLastEventTime(cameraID string) error {
	query := "UPDATE dorm_camera SET last_event_time = NOW() WHERE camera_id = ?"
	_, err := r.DB.Exec(query, cameraID)
	if err != nil {
		return fmt.Errorf("update camera last event time: %w", err)
	}
	return nil
}

// UpdateHealthCheck updates the last_health_check timestamp.
func (r *CameraRepository) UpdateHealthCheck(cameraID string) error {
	query := "UPDATE dorm_camera SET last_health_check = NOW() WHERE camera_id = ?"
	_, err := r.DB.Exec(query, cameraID)
	if err != nil {
		return fmt.Errorf("update camera health check: %w", err)
	}
	return nil
}

// FindWithPagination paginates cameras, filtered by optional building.
func (r *CameraRepository) FindWithPagination(building string, page, size int) ([]entity.DormCamera, int64, error) {
	where := ""
	var args []interface{}
	if building != "" {
		where = "building = ?"
		args = append(args, building)
	}
	return r.BaseRepository.FindWithPagination(where, args, "camera_id ASC", page, size)
}
