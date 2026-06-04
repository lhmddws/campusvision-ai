package service

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"go.uber.org/zap"
)

// AlertService handles alert record CRUD and acknowledgement.
type AlertService struct {
	alertRepo      *repository.AlertRepository
	strangerRepo   *repository.StrangerRecordRepository
	logger         *zap.Logger
}

// NewAlertService creates a new AlertService.
func NewAlertService(
	alertRepo *repository.AlertRepository,
	strangerRepo *repository.StrangerRecordRepository,
	logger *zap.Logger,
) *AlertService {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	return &AlertService{
		alertRepo:    alertRepo,
		strangerRepo: strangerRepo,
		logger:       logger,
	}
}

func (s *AlertService) log() *zap.Logger {
	if s.logger == nil {
		return zap.NewNop()
	}
	return s.logger
}

// GetAlerts returns a paginated list of alerts with optional filters.
func (s *AlertService) GetAlerts(building string, alertType string, acknowledged *bool, page, size int) ([]entity.DormAlert, int64, error) {
	return s.alertRepo.FindWithPagination(serviceCtx(), building, alertType, acknowledged, nil, nil, page, size)
}

// AcknowledgeAlert marks an alert as resolved/acknowledged.
func (s *AlertService) AcknowledgeAlert(id int64) error {
	alert, err := s.alertRepo.FindByID(serviceCtx(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("find alert: %w", err)
	}

	s.log().Info("acknowledging alert", zap.Int64("id", id), zap.String("type", alert.AlertType))
	return s.alertRepo.ResolveAlert(serviceCtx(), id)
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

	return s.alertRepo.Count(serviceCtx(), where, args...)
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
