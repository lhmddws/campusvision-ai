package service

import (
	"database/sql"
	"fmt"
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

// HandleAttendance processes an attendance event (entry/exit).
// This is a skeleton implementation matching the Java stub.
// TODO: Implement full attendance handling with status updates and alerts.
func (s *RecordService) HandleAttendance(record dto.FaceEventMessage) error {
	log.Printf("[RecordService] Handling attendance for studentId=%s, eventType=%s",
		record.StudentID, record.EventType)

	// Build event log entity
	event := &entity.DormEventLog{
		EventType:  record.EventType,
		StudentID:  toNullString(record.StudentID),
		IsStranger: record.IsStranger,
		Confidence: toNullFloat64(record.Confidence),
		Timestamp:  time.UnixMilli(record.Timestamp),
		CreatedAt:  time.Now(),
	}
	if record.CameraID != "" {
		event.CameraID = toNullString(record.CameraID)
	}
	if record.SnapshotPath != "" {
		event.SnapshotPath = toNullString(record.SnapshotPath)
	}

	if _, err := s.eventLogRepo.Create(event); err != nil {
		return fmt.Errorf("create event log: %w", err)
	}

	return nil
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
func (s *RecordService) GetEvents(query dto.EventQueryDTO) ([]entity.DormEventLog, int64, error) {
	return s.eventLogRepo.FindWithPagination(
		query.BuildingID,
		query.CameraID,
		query.EventType,
		query.StudentID,
		query.StartTime,
		query.EndTime,
		query.Page,
		query.Size,
	)
}

func toNullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: f, Valid: f != 0}
}
