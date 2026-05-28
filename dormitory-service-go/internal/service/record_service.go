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
	buildingRepo *repository.BuildingRepository
}

// NewRecordService creates a new RecordService.
func NewRecordService(
	eventLogRepo *repository.EventLogRepository,
	studentRepo *repository.StudentRepository,
	buildingRepo *repository.BuildingRepository,
) *RecordService {
	return &RecordService{
		eventLogRepo: eventLogRepo,
		studentRepo:  studentRepo,
		buildingRepo: buildingRepo,
	}
}

func (s *RecordService) formatDate(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02")
}

// GetAttendanceStats returns aggregated attendance statistics.
func (s *RecordService) GetAttendanceStats(buildingId int64, startDate, endDate time.Time) dto.AttendanceStatsDTO {
	ctx := context.Background()

	building, err := s.buildingRepo.FindByID(ctx, buildingId)
	if err != nil {
		log.Printf("[RecordService] Building not found for id=%d: %v", buildingId, err)
		return dto.AttendanceStatsDTO{}
	}

	startStr := s.formatDate(startDate)
	endStr := s.formatDate(endDate)

	total, err := s.studentRepo.CountActiveByBuilding(ctx, building.Code)
	if err != nil {
		log.Printf("[RecordService] Error counting active students: %v", err)
		return dto.AttendanceStatsDTO{}
	}

	present, err := s.eventLogRepo.CountDistinctPresentStudents(ctx, building.Code, startStr, endStr)
	if err != nil {
		log.Printf("[RecordService] Error counting present students: %v", err)
		return dto.AttendanceStatsDTO{}
	}

	stranger, err := s.eventLogRepo.CountDistinctStrangers(ctx, building.Code, startStr, endStr)
	if err != nil {
		log.Printf("[RecordService] Error counting strangers: %v", err)
		return dto.AttendanceStatsDTO{}
	}

	var rate float64
	if total > 0 {
		rate = float64(present) / float64(total) * 100
	}

	return dto.AttendanceStatsDTO{
		Total:    total,
		Present:  present,
		Absent:   total - present,
		Late:     0,
		Stranger: stranger,
		Rate:     rate,
	}
}

// GetDailySummary returns daily attendance summaries for a date range.
func (s *RecordService) GetDailySummary(buildingId int64, startDate, endDate time.Time) []dto.DailySummaryDTO {
	ctx := context.Background()

	building, err := s.buildingRepo.FindByID(ctx, buildingId)
	if err != nil {
		log.Printf("[RecordService] Building not found for id=%d: %v", buildingId, err)
		return []dto.DailySummaryDTO{}
	}

	startStr := s.formatDate(startDate)
	endStr := s.formatDate(endDate)

	total, err := s.studentRepo.CountActiveByBuilding(ctx, building.Code)
	if err != nil {
		log.Printf("[RecordService] Error counting active students: %v", err)
		return []dto.DailySummaryDTO{}
	}

	dailyCounts, err := s.eventLogRepo.GetDailyPresentCounts(ctx, building.Code, startStr, endStr)
	if err != nil {
		log.Printf("[RecordService] Error getting daily present counts: %v", err)
		return []dto.DailySummaryDTO{}
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	start, _ := time.ParseInLocation("2006-01-02", startStr, loc)
	end, _ := time.ParseInLocation("2006-01-02", endStr, loc)

	countMap := make(map[string]int64)
	for _, dc := range dailyCounts {
		countMap[dc.Date] = dc.PresentCount
	}

	summaries := make([]dto.DailySummaryDTO, 0)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		present := countMap[dateStr]

		var rate float64
		if total > 0 {
			rate = float64(present) / float64(total) * 100
		}

		summaries = append(summaries, dto.DailySummaryDTO{
			Date:         dateStr,
			BuildingName: building.Name,
			CheckinRate:  rate,
		})
	}

	return summaries
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

// GetInspectionList returns per-room inspection roster for a building on today's date.
// Today's date is computed in Asia/Shanghai timezone; no CURDATE() is used in SQL.
func (s *RecordService) GetInspectionList(ctx context.Context, building string) ([]dto.InspectionRoomDTO, error) {
	if building == "" {
		return []dto.InspectionRoomDTO{}, nil
	}

	today := getTodayDateShanghai()

	rows, err := s.studentRepo.FindInspectionList(ctx, building, today)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []dto.InspectionRoomDTO{}, nil
	}

	roomKey := func(r repository.InspectionRow) string { return r.Building + "|" + r.Room }
	grouped := make(map[string][]repository.InspectionRow)
	var keys []string
	for _, r := range rows {
		k := roomKey(r)
		if _, ok := grouped[k]; !ok {
			keys = append(keys, k)
		}
		grouped[k] = append(grouped[k], r)
	}

	result := make([]dto.InspectionRoomDTO, 0, len(keys))
	for _, k := range keys {
		students := grouped[k]
		roomDTO := dto.InspectionRoomDTO{
			Building: students[0].Building,
			Room:     students[0].Room,
			Students: make([]dto.InspectionStudentDTO, 0, len(students)),
		}
		for _, s := range students {
			roomDTO.TotalStudents++
			if s.TodayStatus == "unknown" {
				roomDTO.UnknownCount++
			} else {
				roomDTO.PresentCount++
			}
			roomDTO.Students = append(roomDTO.Students, dto.InspectionStudentDTO{
				StudentID:   s.StudentID,
				StudentName: s.StudentName,
				Status:      s.TodayStatus,
			})
		}
		result = append(result, roomDTO)
	}

	return result, nil
}

func toNullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

func getTodayDateShanghai() string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}
