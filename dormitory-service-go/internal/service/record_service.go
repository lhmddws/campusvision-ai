package service

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
)

// RecordService handles attendance records and event log queries.
type RecordService struct {
	eventLogRepo *repository.EventLogRepository
	studentRepo  *repository.StudentRepository
}

// NewRecordService creates a new RecordService.
func NewRecordService(
	eventLogRepo *repository.EventLogRepository,
	studentRepo *repository.StudentRepository,
) *RecordService {
	return &RecordService{
		eventLogRepo: eventLogRepo,
		studentRepo:  studentRepo,
	}
}

// GetAttendanceStats returns aggregated attendance statistics.
// This is a skeleton implementation matching the Java stub.
// TODO: Implement real aggregation logic.
func (s *RecordService) GetAttendanceStats(buildingId int64, startDate, endDate time.Time) dto.AttendanceStatsDTO {
	log.Printf("[RecordService] Getting attendance stats for buildingId=%d", buildingId)
	return dto.AttendanceStatsDTO{
		Total:    0,
		Present:  0,
		Absent:   0,
		Late:     0,
		Stranger: 0,
		Rate:     0.0,
	}
}

// GetDailySummary returns daily attendance summaries for a date range.
// This is a skeleton implementation matching the Java stub.
// TODO: Implement real daily summary logic.
func (s *RecordService) GetDailySummary(buildingId int64, startDate, endDate time.Time) []dto.DailySummaryDTO {
	log.Printf("[RecordService] Getting daily summary for buildingId=%d", buildingId)
	return []dto.DailySummaryDTO{}
}

// GetEvents returns paginated event logs with optional filters.
// This is a skeleton matching the Java stub that used getEvents(EventQueryDTO, Pageable).
func (s *RecordService) GetEvents(ctx context.Context, query dto.EventQueryDTO) ([]entity.DormEventLog, int64, error) {
	return s.eventLogRepo.FindWithPagination(
		ctx,
		query.Building,
		query.CameraID,
		query.EventType,
		query.StudentID,
		query.StartTime,
		query.EndTime,
		query.Page,
		query.Size,
	)
}

func toNullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}
