package service

import (
	"context"
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"go.uber.org/zap"
)

// RecordService handles attendance records and event log queries.
type RecordService struct {
	eventLogRepo *repository.EventLogRepository
	studentRepo  *repository.StudentRepository
	buildingRepo *repository.BuildingRepository
	logger       *zap.Logger
}

// NewRecordService creates a new RecordService.
func NewRecordService(
	eventLogRepo *repository.EventLogRepository,
	studentRepo *repository.StudentRepository,
	buildingRepo *repository.BuildingRepository,
	logger *zap.Logger,
) *RecordService {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	return &RecordService{
		eventLogRepo: eventLogRepo,
		studentRepo:  studentRepo,
		buildingRepo: buildingRepo,
		logger:       logger,
	}
}

func (s *RecordService) log() *zap.Logger {
	if s.logger == nil {
		return zap.NewNop()
	}
	return s.logger
}

func (s *RecordService) formatDate(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02")
}

// GetAttendanceStats returns aggregated attendance statistics.
func (s *RecordService) GetAttendanceStats(ctx context.Context, buildingId int64, startDate, endDate time.Time) dto.AttendanceStatsDTO {
	building, err := s.buildingRepo.FindByID(ctx, buildingId)
	if err != nil {
		s.log().Warn("building not found", zap.Int64("id", buildingId), zap.Error(err))
		return dto.AttendanceStatsDTO{}
	}

	startStr := s.formatDate(startDate)
	endStr := s.formatDate(endDate)

	total, err := s.studentRepo.CountActiveByBuilding(ctx, building.Code)
	if err != nil {
		s.log().Warn("error counting active students", zap.Error(err))
		return dto.AttendanceStatsDTO{}
	}

	present, err := s.eventLogRepo.CountDistinctPresentStudents(ctx, building.Code, startStr, endStr)
	if err != nil {
		s.log().Warn("error counting present students", zap.Error(err))
		return dto.AttendanceStatsDTO{}
	}

	stranger, err := s.eventLogRepo.CountDistinctStrangers(ctx, building.Code, startStr, endStr)
	if err != nil {
		s.log().Warn("error counting strangers", zap.Error(err))
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
func (s *RecordService) GetDailySummary(ctx context.Context, buildingId int64, startDate, endDate time.Time) []dto.DailySummaryDTO {
	building, err := s.buildingRepo.FindByID(ctx, buildingId)
	if err != nil {
		s.log().Warn("building not found", zap.Int64("id", buildingId), zap.Error(err))
		return []dto.DailySummaryDTO{}
	}

	startStr := s.formatDate(startDate)
	endStr := s.formatDate(endDate)

	total, err := s.studentRepo.CountActiveByBuilding(ctx, building.Code)
	if err != nil {
		s.log().Warn("error counting active students", zap.Error(err))
		return []dto.DailySummaryDTO{}
	}

	dailyCounts, err := s.eventLogRepo.GetDailyPresentCounts(ctx, building.Code, startStr, endStr)
	if err != nil {
		s.log().Warn("error getting daily present counts", zap.Error(err))
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

func getTodayDateShanghai() string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}
