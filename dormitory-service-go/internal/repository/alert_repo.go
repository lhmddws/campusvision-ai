package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// AlertRepository handles dorm_alert_record table operations.
type AlertRepository struct {
	*BaseRepository[entity.DormAlert]
}

// NewAlertRepository creates a new AlertRepository.
func NewAlertRepository(db *sqlx.DB) *AlertRepository {
	return &AlertRepository{
		BaseRepository: NewBaseRepository[entity.DormAlert](db, "dorm_alert_record"),
	}
}

// FindByAlertType finds alerts by type.
func (r *AlertRepository) FindByAlertType(alertType string, limit int) ([]entity.DormAlert, error) {
	if limit <= 0 {
		limit = 100
	}
	var alerts []entity.DormAlert
	query := "SELECT * FROM dorm_alert_record WHERE alert_type = ? ORDER BY occurred_at DESC LIMIT ?"
	err := r.DB.Select(&alerts, query, alertType, limit)
	if err != nil {
		return nil, fmt.Errorf("find alerts by type %s: %w", alertType, err)
	}
	return alerts, nil
}

// FindByBuilding finds alerts for a building.
func (r *AlertRepository) FindByBuilding(building string, limit int) ([]entity.DormAlert, error) {
	if limit <= 0 {
		limit = 100
	}
	var alerts []entity.DormAlert
	query := "SELECT * FROM dorm_alert_record WHERE building = ? ORDER BY occurred_at DESC LIMIT ?"
	err := r.DB.Select(&alerts, query, building, limit)
	if err != nil {
		return nil, fmt.Errorf("find alerts by building %s: %w", building, err)
	}
	return alerts, nil
}

// FindUnresolved finds all unresolved alerts.
func (r *AlertRepository) FindUnresolved() ([]entity.DormAlert, error) {
	var alerts []entity.DormAlert
	query := "SELECT * FROM dorm_alert_record WHERE is_resolved = 0 ORDER BY occurred_at DESC"
	err := r.DB.Select(&alerts, query)
	if err != nil {
		return nil, fmt.Errorf("find unresolved alerts: %w", err)
	}
	return alerts, nil
}

// ResolveAlert marks an alert as resolved.
func (r *AlertRepository) ResolveAlert(id int64) error {
	query := "UPDATE dorm_alert_record SET is_resolved = 1 WHERE id = ?"
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("resolve alert %d: %w", id, err)
	}
	return nil
}

// FindWithPagination paginates alerts with filters.
func (r *AlertRepository) FindWithPagination(
	building string,
	alertType string,
	acknowledged *bool,
	startDate, endDate *time.Time,
	page, size int,
) ([]entity.DormAlert, int64, error) {
	where := ""
	var args []interface{}
	conditions := []string{}

	if building != "" {
		conditions = append(conditions, "building = ?")
		args = append(args, building)
	}
	if alertType != "" {
		conditions = append(conditions, "alert_type = ?")
		args = append(args, alertType)
	}
	if acknowledged != nil {
		conditions = append(conditions, "is_resolved = ?")
		args = append(args, *acknowledged)
	}
	if startDate != nil {
		conditions = append(conditions, "occurred_at >= ?")
		args = append(args, *startDate)
	}
	if endDate != nil {
		conditions = append(conditions, "occurred_at <= ?")
		args = append(args, *endDate)
	}

	if len(conditions) > 0 {
		where = conditions[0]
		for i := 1; i < len(conditions); i++ {
			where += " AND " + conditions[i]
		}
	}

	return r.BaseRepository.FindWithPagination(where, args, "occurred_at DESC", page, size)
}
