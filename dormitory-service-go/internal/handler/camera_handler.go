package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

// CameraHandler handles HTTP requests for /sims/dorm/cameras.
type CameraHandler struct {
	svc *service.CameraService
}

// NewCameraHandler creates a new CameraHandler.
func NewCameraHandler(svc *service.CameraService) *CameraHandler {
	return &CameraHandler{svc: svc}
}

// RegisterCamera    POST /sims/dorm/cameras
func (h *CameraHandler) RegisterCamera(c *gin.Context) {
	var req dto.CameraCreateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	cam, err := h.svc.RegisterCamera(req)
	if err != nil {
		if errors.Is(err, service.ErrCameraLimitExceeded) {
			Error(c, http.StatusBadRequest, "Camera limit exceeded (max 50)")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to register camera: "+err.Error())
		return
	}

	Created(c, cam)
}

// UpdateCamera    PUT /sims/dorm/cameras/:id
func (h *CameraHandler) UpdateCamera(c *gin.Context) {
	cameraID := c.Param("id")
	if cameraID == "" {
		Error(c, http.StatusBadRequest, "Camera ID is required")
		return
	}

	var req dto.CameraUpdateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	cam, err := h.svc.UpdateCamera(cameraID, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Camera not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to update camera: "+err.Error())
		return
	}

	Success(c, gin.H{
		"camera_id": cam.CameraID,
		"updated":   true,
	})
}

// GetCameras    GET /sims/dorm/cameras
func (h *CameraHandler) GetCameras(c *gin.Context) {
	building := c.Query("building")

	cameras, err := h.svc.GetCameras(building)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to list cameras: "+err.Error())
		return
	}

	Success(c, cameras)
}

// GetCamera    GET /sims/dorm/cameras/:id
func (h *CameraHandler) GetCamera(c *gin.Context) {
	cameraID := c.Param("id")

	cam, err := h.svc.GetByCameraID(cameraID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Camera not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to get camera: "+err.Error())
		return
	}

	Success(c, cam)
}

// GetCameraStatus    GET /sims/dorm/cameras/:id/status
func (h *CameraHandler) GetCameraStatus(c *gin.Context) {
	cameraID := c.Param("id")

	cam, err := h.svc.GetByCameraID(cameraID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Camera not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to get camera: "+err.Error())
		return
	}

	status := gin.H{
		"camera_id": cam.CameraID,
		"status":    cam.Status,
		"enabled":   cam.Enabled,
	}
	if cam.LastHealthCheck.Valid {
		status["last_health_check"] = cam.LastHealthCheck.Time
	}

	Success(c, status)
}

// GetCamerasStatus    GET /sims/dorm/cameras/status (with ?building=)
func (h *CameraHandler) GetCamerasStatus(c *gin.Context) {
	building := c.Query("building")

	result, err := h.svc.GetCameraStatus(building)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to get status: "+err.Error())
		return
	}

	Success(c, result)
}

// DeleteCamera    DELETE /sims/dorm/cameras/:id
func (h *CameraHandler) DeleteCamera(c *gin.Context) {
	cameraID := c.Param("id")
	if cameraID == "" {
		Error(c, http.StatusBadRequest, "Camera ID is required")
		return
	}

	if err := h.svc.DeleteCamera(cameraID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Camera not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to delete camera: "+err.Error())
		return
	}

	Success(c, gin.H{
		"camera_id": cameraID,
		"deleted":   true,
	})
}

// HealthCheck    POST /sims/dorm/cameras/:id/health-check
func (h *CameraHandler) HealthCheck(c *gin.Context) {
	cameraID := c.Param("id")

	if err := h.svc.HealthCheck(cameraID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			Error(c, http.StatusNotFound, "Camera not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Health check failed: "+err.Error())
		return
	}

	Success(c, gin.H{
		"camera_id": cameraID,
		"checked":   true,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// QuerySnapshots    GET /sims/dorm/cameras/:id/snapshots
func (h *CameraHandler) QuerySnapshots(c *gin.Context) {
	cameraID := c.Param("id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	events, total, err := h.svc.QuerySnapshots(cameraID, time.Time{}, time.Time{}, page, size)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to query snapshots: "+err.Error())
		return
	}

	PageResult(c, events, total, page, size)
}
