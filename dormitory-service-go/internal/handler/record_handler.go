package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

// RecordHandler handles HTTP requests for /sims/dorm/records.
type RecordHandler struct {
	svc *service.RecordService
}

// NewRecordHandler creates a new RecordHandler.
func NewRecordHandler(svc *service.RecordService) *RecordHandler {
	return &RecordHandler{svc: svc}
}

// HandleAttendance    POST /sims/dorm/records/attendance
func (h *RecordHandler) HandleAttendance(c *gin.Context) {
	var msg dto.FaceEventMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		Error(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.svc.HandleAttendance(msg); err != nil {
		Error(c, http.StatusInternalServerError, "Failed to handle attendance: "+err.Error())
		return
	}

	Success(c, nil)
}

// GetAttendanceStats    GET /sims/dorm/records/attendance/stats
func (h *RecordHandler) GetAttendanceStats(c *gin.Context) {
	buildingID, _ := strconv.ParseInt(c.Query("building_id"), 10, 64)
	startDate, _ := time.Parse("2006-01-02", c.DefaultQuery("start_date", "2000-01-01"))
	endDate, _ := time.Parse("2006-01-02", c.DefaultQuery("end_date", "2099-12-31"))

	stats := h.svc.GetAttendanceStats(buildingID, startDate, endDate)
	Success(c, stats)
}

// GetDailySummary    GET /sims/dorm/records/attendance/daily-summary
func (h *RecordHandler) GetDailySummary(c *gin.Context) {
	buildingID, _ := strconv.ParseInt(c.Query("building_id"), 10, 64)
	startDate, _ := time.Parse("2006-01-02", c.DefaultQuery("start_date", "2000-01-01"))
	endDate, _ := time.Parse("2006-01-02", c.DefaultQuery("end_date", "2099-12-31"))

	summaries := h.svc.GetDailySummary(buildingID, startDate, endDate)
	Success(c, summaries)
}

// GetEvents    GET /sims/dorm/records/events
func (h *RecordHandler) GetEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	buildingID, _ := strconv.ParseInt(c.Query("building_id"), 10, 64)

	var startTime, endTime *time.Time
	if st := c.Query("start_time"); st != "" {
		t, err := time.Parse(time.RFC3339, st)
		if err == nil {
			startTime = &t
		}
	}
	if et := c.Query("end_time"); et != "" {
		t, err := time.Parse(time.RFC3339, et)
		if err == nil {
			endTime = &t
		}
	}

	query := dto.EventQueryDTO{
		BuildingID: buildingID,
		CameraID:   c.Query("camera_id"),
		EventType:  c.Query("event_type"),
		StudentID:  c.Query("student_id"),
		StartTime:  startTime,
		EndTime:    endTime,
		Page:       page,
		Size:       size,
	}

	events, total, err := h.svc.GetEvents(query)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to query events: "+err.Error())
		return
	}

	PageResult(c, events, total, page, size)
}
