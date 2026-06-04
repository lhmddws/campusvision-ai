package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockRecordService(t *testing.T) (sqlmock.Sqlmock, *RecordService) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "mysql")
	t.Cleanup(func() { _ = db.Close() })

	svc := &RecordService{
		eventLogRepo: repository.NewEventLogRepository(sqlxDB),
		studentRepo:  repository.NewStudentRepository(sqlxDB),
		buildingRepo: repository.NewBuildingRepository(sqlxDB),
	}
	return mock, svc
}

func expectBuildingByID(mock sqlmock.Sqlmock, id int64, code, name string) {
	rows := sqlmock.NewRows([]string{"id", "code", "name", "created_at"}).
		AddRow(id, code, name, time.Now())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_building WHERE id = ? LIMIT 1")).
		WithArgs(id).
		WillReturnRows(rows)
}

func expectBuildingByIDError(mock sqlmock.Sqlmock, id int64) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_building WHERE id = ? LIMIT 1")).
		WithArgs(id).
		WillReturnError(assert.AnError)
}

func TestGetInspectionList_EmptyBuilding_ReturnsEmptySlice(t *testing.T) {
	_, svc := newMockRecordService(t)

	result, err := svc.GetInspectionList(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetInspectionList_NormalCase_ReturnsGroupedRooms(t *testing.T) {
	mock, svc := newMockRecordService(t)

	today := time.Now().Format("2006-01-02")
	rows := sqlmock.NewRows([]string{"building", "room", "student_id", "student_name", "today_status"}).
		AddRow("A", "101", "2024001", "张三", "entry").
		AddRow("A", "101", "2024002", "李四", "entry").
		AddRow("A", "101", "2024003", "王五", "exit").
		AddRow("A", "102", "2024004", "赵六", "entry")

	mock.ExpectQuery("SELECT s.building").
		WithArgs(today, "A").
		WillReturnRows(rows)

	result, err := svc.GetInspectionList(context.Background(), "A")
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Room 101: 3 students, 3 present (entry/exit count as present), 0 unknown
	assert.Equal(t, "A", result[0].Building)
	assert.Equal(t, "101", result[0].Room)
	assert.Equal(t, 3, result[0].TotalStudents)
	assert.Equal(t, 3, result[0].PresentCount)
	assert.Equal(t, 0, result[0].UnknownCount)
	assert.Len(t, result[0].Students, 3)

	// Room 102: 1 student, 1 present
	assert.Equal(t, "A", result[1].Building)
	assert.Equal(t, "102", result[1].Room)
	assert.Equal(t, 1, result[1].TotalStudents)
	assert.Equal(t, 1, result[1].PresentCount)
	assert.Equal(t, 0, result[1].UnknownCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetInspectionList_UnknownStatus_CountsCorrectly(t *testing.T) {
	mock, svc := newMockRecordService(t)

	today := time.Now().Format("2006-01-02")
	rows := sqlmock.NewRows([]string{"building", "room", "student_id", "student_name", "today_status"}).
		AddRow("B", "201", "2024010", "孙七", "unknown").
		AddRow("B", "201", "2024011", "周八", "entry").
		AddRow("B", "201", "2024012", "吴九", "unknown")

	mock.ExpectQuery("SELECT s.building").
		WithArgs(today, "B").
		WillReturnRows(rows)

	result, err := svc.GetInspectionList(context.Background(), "B")
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, "B", result[0].Building)
	assert.Equal(t, "201", result[0].Room)
	assert.Equal(t, 3, result[0].TotalStudents)
	assert.Equal(t, 1, result[0].PresentCount)
	assert.Equal(t, 2, result[0].UnknownCount)

	statuses := make(map[string]string)
	for _, s := range result[0].Students {
		statuses[s.StudentID] = s.Status
	}
	assert.Equal(t, "unknown", statuses["2024010"])
	assert.Equal(t, "entry", statuses["2024011"])
	assert.Equal(t, "unknown", statuses["2024012"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetInspectionList_NoStudents_ReturnsEmptySlice(t *testing.T) {
	mock, svc := newMockRecordService(t)

	today := time.Now().Format("2006-01-02")
	rows := sqlmock.NewRows([]string{"building", "room", "student_id", "student_name", "today_status"})

	mock.ExpectQuery("SELECT s.building").
		WithArgs(today, "C").
		WillReturnRows(rows)

	result, err := svc.GetInspectionList(context.Background(), "C")
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- GetAttendanceStats ----------

func TestGetAttendanceStats_ReturnsZeroWhenBuildingNotFound(t *testing.T) {
	mock, svc := newMockRecordService(t)
	expectBuildingByIDError(mock, 1)

	stats := svc.GetAttendanceStats(1, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, int64(0), stats.Total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAttendanceStats_ReturnsCorrectStats(t *testing.T) {
	mock, svc := newMockRecordService(t)
	expectBuildingByID(mock, 1, "A", "Building A")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_student_assignment WHERE building = ? AND active = 1")).
		WithArgs("A").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(100))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT student_id) FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND event_type = 'entry'")).
		WithArgs("A", "2026-05-01", "2026-05-28").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(DISTINCT student_id)"}).AddRow(80))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT student_id) FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND is_stranger = 1")).
		WithArgs("A", "2026-05-01", "2026-05-28").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(DISTINCT student_id)"}).AddRow(3))

	startDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	stats := svc.GetAttendanceStats(1, startDate, endDate)

	assert.Equal(t, int64(100), stats.Total)
	assert.Equal(t, int64(80), stats.Present)
	assert.Equal(t, int64(20), stats.Absent)
	assert.Equal(t, int64(3), stats.Stranger)
	assert.InDelta(t, 80.0, stats.Rate, 0.01)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAttendanceStats_DivisionByZeroReturnsZeroRate(t *testing.T) {
	mock, svc := newMockRecordService(t)
	expectBuildingByID(mock, 1, "B", "Building B")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_student_assignment WHERE building = ? AND active = 1")).
		WithArgs("B").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT student_id) FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND event_type = 'entry'")).
		WithArgs("B", "2026-05-01", "2026-05-28").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(DISTINCT student_id)"}).AddRow(0))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT student_id) FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND is_stranger = 1")).
		WithArgs("B", "2026-05-01", "2026-05-28").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(DISTINCT student_id)"}).AddRow(0))

	startDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	stats := svc.GetAttendanceStats(1, startDate, endDate)

	assert.Equal(t, int64(0), stats.Total)
	assert.Equal(t, int64(0), stats.Present)
	assert.Equal(t, 0.0, stats.Rate)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- GetDailySummary ----------

func TestGetDailySummary_ReturnsEmptyWhenBuildingNotFound(t *testing.T) {
	mock, svc := newMockRecordService(t)
	expectBuildingByIDError(mock, 1)

	summaries := svc.GetDailySummary(1, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC))
	assert.Empty(t, summaries)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDailySummary_ReturnsDailyRowsForDateRange(t *testing.T) {
	mock, svc := newMockRecordService(t)
	expectBuildingByID(mock, 1, "A", "Building A")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_student_assignment WHERE building = ? AND active = 1")).
		WithArgs("A").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(100))

	rows := sqlmock.NewRows([]string{"date", "present_count"}).
		AddRow("2026-05-01", 80).
		AddRow("2026-05-02", 75).
		AddRow("2026-05-03", 90)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DATE(timestamp) AS date, COUNT(DISTINCT student_id) AS present_count FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND event_type = 'entry' GROUP BY DATE(timestamp) ORDER BY date")).
		WithArgs("A", "2026-05-01", "2026-05-03").
		WillReturnRows(rows)

	startDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	summaries := svc.GetDailySummary(1, startDate, endDate)

	require.Len(t, summaries, 3)

	assert.Equal(t, "2026-05-01", summaries[0].Date)
	assert.Equal(t, "Building A", summaries[0].BuildingName)
	assert.InDelta(t, 80.0, summaries[0].CheckinRate, 0.01)

	assert.Equal(t, "2026-05-02", summaries[1].Date)
	assert.InDelta(t, 75.0, summaries[1].CheckinRate, 0.01)

	assert.Equal(t, "2026-05-03", summaries[2].Date)
	assert.InDelta(t, 90.0, summaries[2].CheckinRate, 0.01)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDailySummary_FillsMissingDatesWithZeroRate(t *testing.T) {
	mock, svc := newMockRecordService(t)
	expectBuildingByID(mock, 1, "C", "Building C")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_student_assignment WHERE building = ? AND active = 1")).
		WithArgs("C").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(50))

	rows := sqlmock.NewRows([]string{"date", "present_count"}).
		AddRow("2026-05-01", 40)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DATE(timestamp) AS date, COUNT(DISTINCT student_id) AS present_count FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND event_type = 'entry' GROUP BY DATE(timestamp) ORDER BY date")).
		WithArgs("C", "2026-05-01", "2026-05-03").
		WillReturnRows(rows)

	startDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	summaries := svc.GetDailySummary(1, startDate, endDate)

	require.Len(t, summaries, 3)

	assert.Equal(t, "2026-05-01", summaries[0].Date)
	assert.InDelta(t, 80.0, summaries[0].CheckinRate, 0.01)

	assert.Equal(t, "2026-05-02", summaries[1].Date)
	assert.InDelta(t, 0.0, summaries[1].CheckinRate, 0.01)

	assert.Equal(t, "2026-05-03", summaries[2].Date)
	assert.InDelta(t, 0.0, summaries[2].CheckinRate, 0.01)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDailySummary_DivisionByZeroReturnsZeroRate(t *testing.T) {
	mock, svc := newMockRecordService(t)
	expectBuildingByID(mock, 1, "D", "Building D")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_student_assignment WHERE building = ? AND active = 1")).
		WithArgs("D").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DATE(timestamp) AS date, COUNT(DISTINCT student_id) AS present_count FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND event_type = 'entry' GROUP BY DATE(timestamp) ORDER BY date")).
		WithArgs("D", "2026-05-01", "2026-05-01").
		WillReturnRows(sqlmock.NewRows([]string{"date", "present_count"}))

	startDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	summaries := svc.GetDailySummary(1, startDate, endDate)

	require.Len(t, summaries, 1)
	assert.Equal(t, "2026-05-01", summaries[0].Date)
	assert.Equal(t, 0.0, summaries[0].CheckinRate)
	assert.NoError(t, mock.ExpectationsWereMet())
}
