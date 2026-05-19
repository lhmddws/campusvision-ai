package service

import (
	"fmt"
	"log"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
)

// ReportService handles nightly report generation and queries.
type ReportService struct {
	nightlyReportRepo *repository.NightlyReportRepository
}

// NewReportService creates a new ReportService.
func NewReportService(nightlyReportRepo *repository.NightlyReportRepository) *ReportService {
	return &ReportService{
		nightlyReportRepo: nightlyReportRepo,
	}
}

// GenerateNightlyReport creates a nightly report for a building on a given date.
// This is a skeleton implementation matching the Java stub.
// TODO: Implement real report generation with attendance aggregation.
func (s *ReportService) GenerateNightlyReport(building, date string) (*entity.DormNightlyReport, error) {
	log.Printf("[ReportService] Generating nightly report for building=%s, date=%s", building, date)

	report := &entity.DormNightlyReport{
		ReportDate:      date,
		Building:        building,
		TotalCount:      0,
		PresentCount:    0,
		AbsentCount:     0,
		LateReturnCount: 0,
		StrangerCount:   0,
		UnknownCount:    0,
		Status:          "COMPLETED",
		TriggerType:     "MANUAL",
	}

	id, err := s.nightlyReportRepo.Create(report)
	if err != nil {
		return nil, fmt.Errorf("create nightly report: %w", err)
	}
	report.ID = id

	log.Printf("[ReportService] Nightly report created: building=%s, date=%s, id=%d", building, date, id)
	return report, nil
}

// GetNightlyReport looks up a report by building and date.
func (s *ReportService) GetNightlyReport(building, date string) (*entity.DormNightlyReport, error) {
	return s.nightlyReportRepo.FindByBuildingAndDate(building, date)
}

// GetReportHistory returns paginated report history for a building within a date range.
func (s *ReportService) GetReportHistory(building, startDate, endDate string, page, size int) ([]entity.DormNightlyReport, int64, error) {
	return s.nightlyReportRepo.FindWithPagination(building, startDate, endDate, page, size)
}
