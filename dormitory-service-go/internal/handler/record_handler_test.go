package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"github.com/sims/campusvision/dormitory-service-go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockRecordHandler creates a RecordHandler backed by sqlmock for handler-level tests.
// Uses real RecordService with mock repositories (sqlmock-based DB) to avoid modifying source code.
func newMockRecordHandler(t *testing.T) (sqlmock.Sqlmock, *RecordHandler) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "mysql")
	t.Cleanup(func() { _ = db.Close() })

	svc := service.NewRecordService(
		repository.NewEventLogRepository(sqlxDB),
		repository.NewStudentRepository(sqlxDB),
		repository.NewBuildingRepository(sqlxDB),
		nil,
	)
	return mock, NewRecordHandler(svc)
}

// recordGinContext creates a gin.Context with an httptest.ResponseRecorder for testing.
// The URL may contain query parameters which gin will parse via c.Query().
func recordGinContext(w *httptest.ResponseRecorder, method, url string) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, url, nil)
	return c
}

// expectBuildingByID mocks a dorm_building lookup returning a single row.
func expectBuildingByID(mock sqlmock.Sqlmock, id int64, code, name string) {
	rows := sqlmock.NewRows([]string{"id", "code", "name", "created_at"}).
		AddRow(id, code, name, time.Now())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_building WHERE id = ? LIMIT 1")).
		WithArgs(id).
		WillReturnRows(rows)
}

// ---------- GetAttendanceStats ----------

func TestRecordHandler_GetAttendanceStats_WithBuildingID(t *testing.T) {
	mock, h := newMockRecordHandler(t)

	// Building lookup succeeds
	expectBuildingByID(mock, 1, "A", "Building A")
	// Count active students
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_student_assignment WHERE building = ? AND active = 1")).
		WithArgs("A").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(100))
	// Count distinct present students (entry events in range)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT student_id) FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND event_type = 'entry'")).
		WithArgs("A", "2026-05-01", "2026-05-28").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(DISTINCT student_id)"}).AddRow(80))
	// Count distinct strangers
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(DISTINCT student_id) FROM dorm_entry_exit_event WHERE building = ? AND DATE(timestamp) BETWEEN ? AND ? AND is_stranger = 1")).
		WithArgs("A", "2026-05-01", "2026-05-28").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(DISTINCT student_id)"}).AddRow(3))

	w := httptest.NewRecorder()
	c := recordGinContext(w, http.MethodGet,
		"/sims/dorm/records/attendance/stats?building_id=1&start_date=2026-05-01&end_date=2026-05-28")

	h.GetAttendanceStats(c)

	// Verify HTTP status
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify {code, message, data} envelope
	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)

	// Verify JSON structure has required fields
	raw := make(map[string]interface{})
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	assert.Contains(t, raw, "code")
	assert.Contains(t, raw, "message")
	assert.Contains(t, raw, "data")

	// Verify attendance stats data
	dataJSON, _ := json.Marshal(resp.Data)
	var stats map[string]interface{}
	_ = json.Unmarshal(dataJSON, &stats)
	assert.Equal(t, float64(100), stats["total"])
	assert.Equal(t, float64(80), stats["present"])
	assert.Equal(t, float64(20), stats["absent"])
	assert.Equal(t, float64(3), stats["stranger"])
	assert.InDelta(t, 80.0, stats["rate"], 0.01)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordHandler_GetAttendanceStats_NoBuildingID(t *testing.T) {
	mock, h := newMockRecordHandler(t)

	// Without building_id, buildingID=0 → building not found → empty stats
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_building WHERE id = ? LIMIT 1")).
		WithArgs(int64(0)).
		WillReturnError(assert.AnError)

	w := httptest.NewRecorder()
	c := recordGinContext(w, http.MethodGet, "/sims/dorm/records/attendance/stats")

	h.GetAttendanceStats(c)

	// Handler still returns 200 OK with zero-valued data (service returns empty DTO)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Message)

	// Verify JSON structure
	raw := make(map[string]interface{})
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	assert.Contains(t, raw, "code")
	assert.Contains(t, raw, "message")
	assert.Contains(t, raw, "data")

	// Data should be zero-valued AttendanceStatsDTO
	dataJSON, _ := json.Marshal(resp.Data)
	var stats map[string]interface{}
	_ = json.Unmarshal(dataJSON, &stats)
	assert.Equal(t, float64(0), stats["total"])
	assert.Equal(t, float64(0), stats["present"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordHandler_GetAttendanceStats_InvalidBuildingID(t *testing.T) {
	_, h := newMockRecordHandler(t)

	w := httptest.NewRecorder()
	c := recordGinContext(w, http.MethodGet,
		"/sims/dorm/records/attendance/stats?building_id=abc")

	h.GetAttendanceStats(c)

	// Invalid query param → 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "invalid building_id", resp.Message)
	assert.Nil(t, resp.Data)
}

// ---------- GetEvents ----------

func TestRecordHandler_GetEvents_WithPagination(t *testing.T) {
	mock, h := newMockRecordHandler(t)

	// COUNT query with filters
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_entry_exit_event WHERE building = ? AND camera_id = ? AND event_type = ?")).
		WithArgs("A", "cam-01", "entry").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))

	// SELECT with filters, ORDER BY timestamp DESC, LIMIT 20 OFFSET 0
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_entry_exit_event WHERE building = ? AND camera_id = ? AND event_type = ? ORDER BY timestamp DESC LIMIT 20 OFFSET 0")).
		WithArgs("A", "cam-01", "entry").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "event_id", "camera_id", "building", "event_type",
			"student_id", "student_name", "is_stranger", "is_processed",
			"confidence", "face_snapshot_url", "timestamp", "created_at",
		}).AddRow(1, "evt-001", "cam-01", "A", "entry",
			"S001", "张三", false, false,
			0.95, nil, now, now))

	w := httptest.NewRecorder()
	c := recordGinContext(w, http.MethodGet,
		"/sims/dorm/records/events?building=A&camera_id=cam-01&event_type=entry&page=1&size=20")

	h.GetEvents(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Message)

	// Verify {code, message, data} structure
	raw := make(map[string]interface{})
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	assert.Contains(t, raw, "code")
	assert.Contains(t, raw, "message")
	assert.Contains(t, raw, "data")

	// Verify PageData structure: items, total, page, size
	dataJSON, _ := json.Marshal(resp.Data)
	var pageData PageData
	err = json.Unmarshal(dataJSON, &pageData)
	require.NoError(t, err)
	assert.Equal(t, int64(1), pageData.Total)
	assert.Equal(t, 1, pageData.Page)
	assert.Equal(t, 20, pageData.Size)
	assert.NotNil(t, pageData.Items)

	// Verify items is a non-empty array
	itemsJSON, _ := json.Marshal(pageData.Items)
	var items []interface{}
	_ = json.Unmarshal(itemsJSON, &items)
	assert.Len(t, items, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordHandler_GetEvents_DefaultPagination(t *testing.T) {
	mock, h := newMockRecordHandler(t)

	// No filters → COUNT(*) without WHERE
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_entry_exit_event")).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

	// No filters → SELECT * without WHERE, defaults (page=1, size=20, timestamp DESC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_entry_exit_event ORDER BY timestamp DESC LIMIT 20 OFFSET 0")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "event_id", "camera_id", "building", "event_type",
			"student_id", "student_name", "is_stranger", "is_processed",
			"confidence", "face_snapshot_url", "timestamp", "created_at",
		}))

	w := httptest.NewRecorder()
	c := recordGinContext(w, http.MethodGet, "/sims/dorm/records/events")

	h.GetEvents(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Verify default pagination values (page=1, size=20)
	dataJSON, _ := json.Marshal(resp.Data)
	var pageData PageData
	err = json.Unmarshal(dataJSON, &pageData)
	require.NoError(t, err)
	assert.Equal(t, int64(0), pageData.Total)
	assert.Equal(t, 1, pageData.Page)   // default page
	assert.Equal(t, 20, pageData.Size)  // default size
	assert.Empty(t, pageData.Items)

	assert.NoError(t, mock.ExpectationsWereMet())
}
