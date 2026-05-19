package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// StrangerRecordRepository handles dorm_stranger_record table operations.
type StrangerRecordRepository struct {
	*BaseRepository[entity.DormStrangerRecord]
}

// NewStrangerRecordRepository creates a new StrangerRecordRepository.
func NewStrangerRecordRepository(db *sqlx.DB) *StrangerRecordRepository {
	return &StrangerRecordRepository{
		BaseRepository: NewBaseRepository[entity.DormStrangerRecord](db, "dorm_stranger_record"),
	}
}

// FindByBuilding finds stranger records for a building.
func (r *StrangerRecordRepository) FindByBuilding(building string, limit int) ([]entity.DormStrangerRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	var records []entity.DormStrangerRecord
	query := "SELECT * FROM dorm_stranger_record WHERE building = ? ORDER BY detected_time DESC LIMIT ?"
	err := r.DB.Select(&records, query, building, limit)
	if err != nil {
		return nil, fmt.Errorf("find stranger records by building %s: %w", building, err)
	}
	return records, nil
}

// FindByStatus finds stranger records by status.
func (r *StrangerRecordRepository) FindByStatus(status string) ([]entity.DormStrangerRecord, error) {
	var records []entity.DormStrangerRecord
	query := "SELECT * FROM dorm_stranger_record WHERE status = ? ORDER BY detected_time DESC"
	err := r.DB.Select(&records, query, status)
	if err != nil {
		return nil, fmt.Errorf("find stranger records by status %s: %w", status, err)
	}
	return records, nil
}

// FindByTimeRange finds stranger records within a time range.
func (r *StrangerRecordRepository) FindByTimeRange(start, end time.Time) ([]entity.DormStrangerRecord, error) {
	var records []entity.DormStrangerRecord
	query := "SELECT * FROM dorm_stranger_record WHERE detected_time >= ? AND detected_time <= ? ORDER BY detected_time DESC"
	err := r.DB.Select(&records, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("find stranger records by time range: %w", err)
	}
	return records, nil
}

// FindWithPagination paginates stranger records.
func (r *StrangerRecordRepository) FindWithPagination(
	building string,
	status string,
	startDate, endDate *time.Time,
	page, size int,
) ([]entity.DormStrangerRecord, int64, error) {
	where := ""
	var args []interface{}
	conditions := []string{}

	if building != "" {
		conditions = append(conditions, "building = ?")
		args = append(args, building)
	}
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if startDate != nil {
		conditions = append(conditions, "detected_time >= ?")
		args = append(args, *startDate)
	}
	if endDate != nil {
		conditions = append(conditions, "detected_time <= ?")
		args = append(args, *endDate)
	}

	if len(conditions) > 0 {
		where = conditions[0]
		for i := 1; i < len(conditions); i++ {
			where += " AND " + conditions[i]
		}
	}

	return r.BaseRepository.FindWithPagination(where, args, "detected_time DESC", page, size)
}
