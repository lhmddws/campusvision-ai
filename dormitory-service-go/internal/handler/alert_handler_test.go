package handler

import (
	"database/sql/driver"
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

// newMockAlertHandler creates an AlertHandler with a go-sqlmock backed database.
func newMockAlertHandler(t *testing.T) (sqlmock.Sqlmock, *AlertHandler) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "mysql")
	t.Cleanup(func() { _ = db.Close() })

	svc := service.NewAlertService(
		repository.NewAlertRepository(sqlxDB),
		repository.NewStrangerRecordRepository(sqlxDB),
		nil,
	)
	return mock, NewAlertHandler(svc)
}

// dormAlertColumns is the full column list for SELECT * FROM dorm_alert_record.
var dormAlertColumns = []string{
	"id", "alert_id", "alert_type", "building", "student_id", "severity",
	"description", "face_snapshot_url", "is_read", "is_resolved", "occurred_at", "created_at",
}

func mockDormAlertRow(id int64, alertType, building, severity string, isResolved bool) []driver.Value {
	return []driver.Value{
		id, "ALT-" + alertType, alertType, building, "STU-" + alertType, severity,
		alertType + " detected", nil, false, isResolved, time.Now(), time.Now(),
	}
}

func TestAlertHandler_GetAlerts(t *testing.T) {
	mock, handler := newMockAlertHandler(t)

	// Expect count query with building and alert_type filters
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_alert_record WHERE building = ? AND alert_type = ?")).
		WithArgs("A", "stranger").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(int64(2)))

	// Expect select query with pagination LIMIT 10 OFFSET 10 (page=2, size=10)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_alert_record WHERE building = ? AND alert_type = ? ORDER BY occurred_at DESC LIMIT 10 OFFSET 10")).
		WithArgs("A", "stranger").
		WillReturnRows(sqlmock.NewRows(dormAlertColumns).
			AddRow(mockDormAlertRow(1, "stranger", "A", "high", false)...).
			AddRow(mockDormAlertRow(2, "stranger", "A", "medium", false)...))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sims/dorm/alerts?building=A&alert_type=stranger&page=2&size=10", nil)

	handler.GetAlerts(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Message)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "data should be a map")
	assert.Equal(t, float64(2), data["total"])
	assert.Equal(t, float64(2), data["page"])
	assert.Equal(t, float64(10), data["size"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok, "items should be an array")
	assert.Len(t, items, 2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAlertHandler_GetAlerts_DefaultPagination(t *testing.T) {
	mock, handler := newMockAlertHandler(t)

	// No WHERE clause — no filters provided
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM dorm_alert_record")).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_alert_record ORDER BY occurred_at DESC LIMIT 20 OFFSET 0")).
		WillReturnRows(sqlmock.NewRows(dormAlertColumns).
			AddRow(mockDormAlertRow(1, "absent", "B", "low", false)...))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sims/dorm/alerts", nil)

	handler.GetAlerts(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Message)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["total"])
	assert.Equal(t, float64(1), data["page"], "default page should be 1")
	assert.Equal(t, float64(20), data["size"], "default size should be 20")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAlertHandler_AcknowledgeAlert(t *testing.T) {
	mock, handler := newMockAlertHandler(t)

	// Expect FindByID to return alert
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_alert_record WHERE id = ? LIMIT 1")).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows(dormAlertColumns).
			AddRow(mockDormAlertRow(123, "stranger", "A", "high", false)...))

	// Expect ResolveAlert update
	mock.ExpectExec(regexp.QuoteMeta("UPDATE dorm_alert_record SET is_resolved = 1 WHERE id = ?")).
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/sims/dorm/alerts/123/acknowledge", nil)
	c.Params = []gin.Param{{Key: "id", Value: "123"}}

	handler.AcknowledgeAlert(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Message)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(123), data["id"])
	assert.Equal(t, true, data["acknowledged"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAlertHandler_AcknowledgeAlert_NotFound(t *testing.T) {
	mock, handler := newMockAlertHandler(t)

	// FindByID returns ErrNoRows -> service returns ErrNotFound -> handler returns 404
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM dorm_alert_record WHERE id = ? LIMIT 1")).
		WithArgs(int64(999)).
		WillReturnError(service.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/sims/dorm/alerts/999/acknowledge", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	handler.AcknowledgeAlert(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "Alert not found", resp.Message)

	assert.NoError(t, mock.ExpectationsWereMet())
}
