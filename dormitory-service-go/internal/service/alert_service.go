package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"go.uber.org/zap"
)

// AlertService handles alert record CRUD and acknowledgement.
type AlertService struct {
	alertRepo    *repository.AlertRepository
	strangerRepo *repository.StrangerRecordRepository
	logger       *zap.Logger
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
func (s *AlertService) GetAlerts(ctx context.Context, building string, alertType string, acknowledged *bool, page, size int) ([]entity.DormAlert, int64, error) {
	return s.alertRepo.FindWithPagination(ctx, building, alertType, acknowledged, nil, nil, page, size)
}

// AcknowledgeAlert marks an alert as resolved/acknowledged.
func (s *AlertService) AcknowledgeAlert(ctx context.Context, id int64) error {
	alert, err := s.alertRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("find alert: %w", err)
	}

	s.log().Info("acknowledging alert", zap.Int64("id", id), zap.String("type", alert.AlertType))
	return s.alertRepo.ResolveAlert(ctx, id)
}

// GetAlertCount returns the count of alerts with optional filters.
func (s *AlertService) GetAlertCount(ctx context.Context, building string, acknowledged *bool) (int64, error) {
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

	return s.alertRepo.Count(ctx, where, args...)
}

// AlertStatsResponse represents alert statistics with additional breakdowns.
type AlertStatsResponse struct {
	Total      int64            `json:"total"`
	Unresolved int64            `json:"unresolved"`
	Unread     int64            `json:"unread"`
	Today      int64            `json:"today"`
	ByType     map[string]int64 `json:"by_type"`
	BySeverity map[string]int64 `json:"by_severity"`
}

// GetAlertStats returns total, unresolved, unread, today counts and breakdowns by type and severity.
func (s *AlertService) GetAlertStats(ctx context.Context, building string) (*AlertStatsResponse, error) {
	total, err := s.GetAlertCount(ctx, building, nil)
	if err != nil {
		return nil, fmt.Errorf("count total: %w", err)
	}

	unresolved := false
	unresolvedCount, err := s.GetAlertCount(ctx, building, &unresolved)
	if err != nil {
		return nil, fmt.Errorf("count unresolved: %w", err)
	}

	unreadQuery := "SELECT COUNT(*) FROM dorm_alert_record WHERE is_read = 0"
	var unreadArgs []interface{}
	if building != "" {
		unreadQuery += " AND building = ?"
		unreadArgs = append(unreadArgs, building)
	}
	var unreadCount int64
	if err := s.alertRepo.DB.GetContext(ctx, &unreadCount, unreadQuery, unreadArgs...); err != nil {
		return nil, fmt.Errorf("count unread: %w", err)
	}

	todayQuery := "SELECT COUNT(*) FROM dorm_alert_record WHERE DATE(created_at) = CURDATE()"
	var todayArgs []interface{}
	if building != "" {
		todayQuery += " AND building = ?"
		todayArgs = append(todayArgs, building)
	}
	var todayCount int64
	if err := s.alertRepo.DB.GetContext(ctx, &todayCount, todayQuery, todayArgs...); err != nil {
		return nil, fmt.Errorf("count today: %w", err)
	}

	typeQuery := "SELECT alert_type, COUNT(*) as count FROM dorm_alert_record"
	var typeArgs []interface{}
	if building != "" {
		typeQuery += " WHERE building = ?"
		typeArgs = append(typeArgs, building)
	}
	typeQuery += " GROUP BY alert_type"
	typeRows, err := s.alertRepo.DB.QueryContext(ctx, typeQuery, typeArgs...)
	if err != nil {
		return nil, fmt.Errorf("query by_type: %w", err)
	}
	defer typeRows.Close()

	byType := make(map[string]int64)
	for typeRows.Next() {
		var alertType string
		var count int64
		if err := typeRows.Scan(&alertType, &count); err != nil {
			return nil, fmt.Errorf("scan by_type row: %w", err)
		}
		byType[alertType] = count
	}
	if err := typeRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate by_type rows: %w", err)
	}

	severityQuery := "SELECT severity, COUNT(*) as count FROM dorm_alert_record"
	var severityArgs []interface{}
	if building != "" {
		severityQuery += " WHERE building = ?"
		severityArgs = append(severityArgs, building)
	}
	severityQuery += " GROUP BY severity"
	severityRows, err := s.alertRepo.DB.QueryContext(ctx, severityQuery, severityArgs...)
	if err != nil {
		return nil, fmt.Errorf("query by_severity: %w", err)
	}
	defer severityRows.Close()

	bySeverity := make(map[string]int64)
	for severityRows.Next() {
		var severity string
		var count int64
		if err := severityRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("scan by_severity row: %w", err)
		}
		bySeverity[severity] = count
	}
	if err := severityRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate by_severity rows: %w", err)
	}

	return &AlertStatsResponse{
		Total:      total,
		Unresolved: unresolvedCount,
		Unread:     unreadCount,
		Today:      todayCount,
		ByType:     byType,
		BySeverity: bySeverity,
	}, nil
}
