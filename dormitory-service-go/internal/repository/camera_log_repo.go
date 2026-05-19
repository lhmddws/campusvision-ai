package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// CameraLogRepository handles dorm_camera_log table operations.
type CameraLogRepository struct {
	*BaseRepository[entity.DormCameraLog]
}

// NewCameraLogRepository creates a new CameraLogRepository.
func NewCameraLogRepository(db *sqlx.DB) *CameraLogRepository {
	return &CameraLogRepository{
		BaseRepository: NewBaseRepository[entity.DormCameraLog](db, "dorm_camera_log"),
	}
}

// FindByCameraID finds camera logs for a specific camera.
func (r *CameraLogRepository) FindByCameraID(cameraID string, limit int) ([]entity.DormCameraLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var logs []entity.DormCameraLog
	query := "SELECT * FROM dorm_camera_log WHERE camera_id = ? ORDER BY created_at DESC LIMIT ?"
	err := r.DB.Select(&logs, query, cameraID, limit)
	if err != nil {
		return nil, fmt.Errorf("find camera logs by camera %s: %w", cameraID, err)
	}
	return logs, nil
}

// FindByBuilding finds camera logs for a building.
func (r *CameraLogRepository) FindByBuilding(building string, limit int) ([]entity.DormCameraLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var logs []entity.DormCameraLog
	query := "SELECT * FROM dorm_camera_log WHERE building = ? ORDER BY created_at DESC LIMIT ?"
	err := r.DB.Select(&logs, query, building, limit)
	if err != nil {
		return nil, fmt.Errorf("find camera logs by building %s: %w", building, err)
	}
	return logs, nil
}

// FindByTimeRange finds camera logs within a time range.
func (r *CameraLogRepository) FindByTimeRange(start, end time.Time, limit int) ([]entity.DormCameraLog, error) {
	if limit <= 0 {
		limit = 1000
	}
	var logs []entity.DormCameraLog
	query := "SELECT * FROM dorm_camera_log WHERE created_at >= ? AND created_at <= ? ORDER BY created_at DESC LIMIT ?"
	err := r.DB.Select(&logs, query, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("find camera logs by time range: %w", err)
	}
	return logs, nil
}

// FindWithPagination paginates camera logs.
func (r *CameraLogRepository) FindWithPagination(
	cameraID, building string,
	page, size int,
) ([]entity.DormCameraLog, int64, error) {
	where := ""
	var args []interface{}
	conditions := []string{}

	if cameraID != "" {
		conditions = append(conditions, "camera_id = ?")
		args = append(args, cameraID)
	}
	if building != "" {
		conditions = append(conditions, "building = ?")
		args = append(args, building)
	}

	if len(conditions) > 0 {
		where = conditions[0]
		for i := 1; i < len(conditions); i++ {
			where += " AND " + conditions[i]
		}
	}

	return r.BaseRepository.FindWithPagination(where, args, "created_at DESC", page, size)
}
