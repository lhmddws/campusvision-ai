package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

// AlertHandler handles HTTP requests for /sims/dorm/alerts.
type AlertHandler struct {
	svc *service.AlertService
}

// NewAlertHandler creates a new AlertHandler.
func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

// GetAlerts    GET /sims/dorm/alerts
func (h *AlertHandler) GetAlerts(c *gin.Context) {
	building := c.Query("building")
	alertType := c.Query("alert_type")

	var acknowledged *bool
	if ack := c.Query("acknowledged"); ack != "" {
		b := ack == "true" || ack == "1"
		acknowledged = &b
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	alerts, total, err := h.svc.GetAlerts(building, alertType, acknowledged, page, size)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to query alerts: "+err.Error())
		return
	}

	PageResult(c, alerts, total, page, size)
}

// AcknowledgeAlert    POST /sims/dorm/alerts/:id/acknowledge
func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "Invalid alert ID")
		return
	}

	if err := h.svc.AcknowledgeAlert(id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Alert not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to acknowledge alert: "+err.Error())
		return
	}

	Success(c, gin.H{
		"id":           id,
		"acknowledged": true,
	})
}

// GetAlertStats    GET /sims/dorm/alerts/stats
func (h *AlertHandler) GetAlertStats(c *gin.Context) {
	building := c.Query("building")

	stats, err := h.svc.GetAlertStats(building)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to get alert stats: "+err.Error())
		return
	}

	Success(c, stats)
}
