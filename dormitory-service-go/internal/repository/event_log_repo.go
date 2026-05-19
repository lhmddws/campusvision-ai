package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// EventLogRepository handles dorm_event_log table operations.
type EventLogRepository struct {
	*BaseRepository[entity.DormEventLog]
}

// NewEventLogRepository creates a new EventLogRepository.
func NewEventLogRepository(db *sqlx.DB) *EventLogRepository {
	return &EventLogRepository{
		BaseRepository: NewBaseRepository[entity.DormEventLog](db, "dorm_event_log"),
	}
}

// FindByCameraID finds events for a specific camera.
func (r *EventLogRepository) FindByCameraID(cameraID string, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_event_log WHERE camera_id = ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.Select(&events, query, cameraID, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by camera %s: %w", cameraID, err)
	}
	return events, nil
}

// FindByStudentID finds events for a specific student.
func (r *EventLogRepository) FindByStudentID(studentID string, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_event_log WHERE student_id = ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.Select(&events, query, studentID, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by student %s: %w", studentID, err)
	}
	return events, nil
}

// FindByEventType finds events by type (entry/exit).
func (r *EventLogRepository) FindByEventType(eventType string, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_event_log WHERE event_type = ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.Select(&events, query, eventType, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by type %s: %w", eventType, err)
	}
	return events, nil
}

// FindByTimeRange finds events within a time range.
func (r *EventLogRepository) FindByTimeRange(start, end time.Time, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 1000
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_event_log WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.Select(&events, query, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by time range: %w", err)
	}
	return events, nil
}

// FindByBuilding finds events for a given building.
func (r *EventLogRepository) FindByBuilding(buildingID int64, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_event_log WHERE building_id = ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.Select(&events, query, buildingID, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by building %d: %w", buildingID, err)
	}
	return events, nil
}

// FindWithPagination paginates events with filters.
func (r *EventLogRepository) FindWithPagination(
	buildingID int64,
	cameraID string,
	eventType string,
	studentID string,
	startTime, endTime *time.Time,
	page, size int,
) ([]entity.DormEventLog, int64, error) {
	where := ""
	var args []interface{}
	conditions := []string{}

	if buildingID > 0 {
		conditions = append(conditions, "building_id = ?")
		args = append(args, buildingID)
	}
	if cameraID != "" {
		conditions = append(conditions, "camera_id = ?")
		args = append(args, cameraID)
	}
	if eventType != "" {
		conditions = append(conditions, "event_type = ?")
		args = append(args, eventType)
	}
	if studentID != "" {
		conditions = append(conditions, "student_id = ?")
		args = append(args, studentID)
	}
	if startTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *startTime)
	}
	if endTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *endTime)
	}

	if len(conditions) > 0 {
		where = conditions[0]
		for i := 1; i < len(conditions); i++ {
			where += " AND " + conditions[i]
		}
	}

	return r.BaseRepository.FindWithPagination(where, args, "timestamp DESC", page, size)
}
