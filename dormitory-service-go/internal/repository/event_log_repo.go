package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// EventLogRepository handles dorm_entry_exit_event table operations.
type EventLogRepository struct {
	*BaseRepository[entity.DormEventLog]
}

// NewEventLogRepository creates a new EventLogRepository.
func NewEventLogRepository(db *sqlx.DB) *EventLogRepository {
	return &EventLogRepository{
		BaseRepository: NewBaseRepository[entity.DormEventLog](db, "dorm_entry_exit_event"),
	}
}

// FindByCameraID finds events for a specific camera.
func (r *EventLogRepository) FindByCameraID(ctx context.Context, cameraID string, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_entry_exit_event WHERE camera_id = ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.SelectContext(ctx, &events, query, cameraID, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by camera %s: %w", cameraID, err)
	}
	return events, nil
}

// FindByStudentID finds events for a specific student.
func (r *EventLogRepository) FindByStudentID(ctx context.Context, studentID string, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_entry_exit_event WHERE student_id = ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.SelectContext(ctx, &events, query, studentID, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by student %s: %w", studentID, err)
	}
	return events, nil
}

// FindByEventType finds events by type (entry/exit).
func (r *EventLogRepository) FindByEventType(ctx context.Context, eventType string, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_entry_exit_event WHERE event_type = ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.SelectContext(ctx, &events, query, eventType, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by type %s: %w", eventType, err)
	}
	return events, nil
}

// FindByTimeRange finds events within a time range.
func (r *EventLogRepository) FindByTimeRange(ctx context.Context, start, end time.Time, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 1000
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_entry_exit_event WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.SelectContext(ctx, &events, query, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by time range: %w", err)
	}
	return events, nil
}

// FindByBuilding finds events for a given building.
func (r *EventLogRepository) FindByBuilding(ctx context.Context, building string, limit int) ([]entity.DormEventLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var events []entity.DormEventLog
	query := "SELECT * FROM dorm_entry_exit_event WHERE building = ? ORDER BY timestamp DESC LIMIT ?"
	err := r.DB.SelectContext(ctx, &events, query, building, limit)
	if err != nil {
		return nil, fmt.Errorf("find events by building '%s': %w", building, err)
	}
	return events, nil
}

// FindWithPagination paginates events with filters.
func (r *EventLogRepository) FindWithPagination(
	ctx context.Context,
	building string,
	cameraID string,
	eventType string,
	studentID string,
	startTime, endTime *time.Time,
	page, size int,
) ([]entity.DormEventLog, int64, error) {
	where := ""
	var args []interface{}
	conditions := []string{}

	if building != "" {
		conditions = append(conditions, "building = ?")
		args = append(args, building)
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

	return r.BaseRepository.FindWithPagination(ctx, where, args, "timestamp DESC", page, size)
}
