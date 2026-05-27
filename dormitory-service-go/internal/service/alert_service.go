package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
)

// AlertService handles alert record CRUD and acknowledgement.
type AlertService struct {
	alertRepo      *repository.AlertRepository
	strangerRepo   *repository.StrangerRecordRepository
}

// NewAlertService creates a new AlertService.
func NewAlertService(
	alertRepo *repository.AlertRepository,
	strangerRepo *repository.StrangerRecordRepository,
) *AlertService {
	return &AlertService{
		alertRepo:    alertRepo,
		strangerRepo: strangerRepo,
	}
}

// GetAlerts returns a paginated list of alerts with optional filters.
func (s *AlertService) GetAlerts(building string, alertType string, acknowledged *bool, page, size int) ([]entity.DormAlert, int64, error) {
	return s.alertRepo.FindWithPagination(context.Background(), building, alertType, acknowledged, nil, nil, page, size)
}

// AcknowledgeAlert marks an alert as resolved/acknowledged.
func (s *AlertService) AcknowledgeAlert(id int64) error {
	alert, err := s.alertRepo.FindByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("find alert: %w", err)
	}

	log.Printf("[AlertService] Acknowledging alert id=%d, type=%s", id, alert.AlertType)
	return s.alertRepo.ResolveAlert(context.Background(), id)
}

// GetAlertCount returns the count of alerts with optional filters.
func (s *AlertService) GetAlertCount(building string, acknowledged *bool) (int64, error) {
	where := ""
	var args []interface{}
	conditions := []string{}

	if building != "" {
		conditions = append(conditions, "building = ?")
		args = append(args, building)
	}
	if acknowledged != nil {
		if *acknowledged {
			conditions = append(conditions, "is_resolved = 1")
		} else {
			conditions = append(conditions, "is_resolved = 0")
		}
	}

	if len(conditions) > 0 {
		where = conditions[0]
		for i := 1; i < len(conditions); i++ {
			where += " AND " + conditions[i]
		}
	}

	return s.alertRepo.Count(context.Background(), where, args...)
}

// GetAlertStats returns total and unresolved alert counts for a building.
func (s *AlertService) GetAlertStats(building string) (map[string]interface{}, error) {
	total, err := s.GetAlertCount(building, nil)
	if err != nil {
		return nil, fmt.Errorf("count total: %w", err)
	}

	unresolved := false
	unresolvedCount, err := s.GetAlertCount(building, &unresolved)
	if err != nil {
		return nil, fmt.Errorf("count unresolved: %w", err)
	}

	return map[string]interface{}{
		"total":      total,
		"unresolved": unresolvedCount,
	}, nil
}

// CreateAlert inserts a new alert record.
func (s *AlertService) CreateAlert(building, alertType, message, details string) (*entity.DormAlert, error) {
	alert := &entity.DormAlert{
		AlertType:  alertType,
		Building:   toNullString(building),
		Severity:   "medium",
		Description: toNullString(message),
		IsRead:     false,
		IsResolved: false,
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	id, err := s.alertRepo.Create(context.Background(), alert)
	if err != nil {
		return nil, fmt.Errorf("create alert: %w", err)
	}
	alert.ID = id

	log.Printf("[AlertService] Created alert id=%d, type=%s", id, alertType)
	return alert, nil
}
