package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

// FaceHandler handles HTTP requests for /sims/dorm/faces.
type FaceHandler struct {
	svc *service.FaceService
}

// NewFaceHandler creates a new FaceHandler.
func NewFaceHandler(svc *service.FaceService) *FaceHandler {
	return &FaceHandler{svc: svc}
}

// Create    POST /sims/dorm/faces
func (h *FaceHandler) Create(c *gin.Context) {
	var req dto.FaceCreateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	face, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateStudentID) {
			Error(c, http.StatusConflict, "student_id already exists")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to create face record: "+err.Error())
		return
	}

	Created(c, face)
}

// Enroll    POST /sims/dorm/faces/enroll
func (h *FaceHandler) Enroll(c *gin.Context) {
	studentID := c.PostForm("student_id")
	name := c.PostForm("name")
	if studentID == "" || name == "" {
		Error(c, http.StatusBadRequest, "student_id and name are required")
		return
	}
	roomNumber := c.PostForm("room_number")

	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		Error(c, http.StatusBadRequest, "photo file is required: "+err.Error())
		return
	}
	defer func() { _ = file.Close() }()

	photoBytes, err := io.ReadAll(file)
	if err != nil {
		Error(c, http.StatusInternalServerError, "failed to read photo: "+err.Error())
		return
	}

	face, err := h.svc.Enroll(c.Request.Context(), dto.FaceEnrollDTO{
		StudentID:  studentID,
		Name:       name,
		RoomNumber: roomNumber,
		PhotoBytes: photoBytes,
	})
	if err != nil {
		if errors.Is(err, service.ErrDuplicateStudentID) {
			Error(c, http.StatusConflict, "student_id already exists")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to enroll face: "+err.Error())
		return
	}

	Created(c, face)
}

// Update    PUT /sims/dorm/faces/:id
func (h *FaceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "Invalid face ID")
		return
	}

	var req dto.FaceUpdateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	face, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrFaceNotFound) {
			Error(c, http.StatusNotFound, "Face record not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to update face record: "+err.Error())
		return
	}

	Success(c, face)
}

// Delete    DELETE /sims/dorm/faces/:id
func (h *FaceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "Invalid face ID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrFaceNotFound) {
			Error(c, http.StatusNotFound, "Face record not found")
			return
		}
		Error(c, http.StatusInternalServerError, "Failed to delete face record: "+err.Error())
		return
	}

	Success(c, gin.H{"id": id, "deleted": true})
}

// List    GET /sims/dorm/faces
func (h *FaceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	faces, total, err := h.svc.List(c.Request.Context(), page, size)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Failed to list face records: "+err.Error())
		return
	}

	PageResult(c, faces, total, page, size)
}

// BatchImport    POST /sims/dorm/faces/batch-import
func (h *FaceHandler) BatchImport(c *gin.Context) {
	var req dto.FaceBatchImportDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	created, duplicates, err := h.svc.BatchImport(c.Request.Context(), req)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Batch import failed: "+err.Error())
		return
	}

	Success(c, gin.H{
		"created":    created,
		"duplicates": duplicates,
		"total":      len(req.Students),
	})
}
