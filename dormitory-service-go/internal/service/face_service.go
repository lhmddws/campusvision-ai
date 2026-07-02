package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"go.uber.org/zap"
)

var ErrFaceNotFound = errors.New("face record not found")
var ErrDuplicateStudentID = errors.New("student_id already exists")

// FaceService handles face record CRUD operations.
type FaceService struct {
	repo              *repository.FaceRepository
	logger            *zap.Logger
	faceRecognitionURL string
	httpClient        *http.Client
	uploadDir         string
}

// NewFaceService creates a new FaceService.
func NewFaceService(repo *repository.FaceRepository, logger *zap.Logger, faceRecognitionURL string) *FaceService {
	return &FaceService{
		repo:               repo,
		logger:             logger,
		faceRecognitionURL: faceRecognitionURL,
		httpClient:         &http.Client{Timeout: 15 * time.Second},
		uploadDir:          "uploads/faces",
	}
}

// Create adds a new face record.
func (s *FaceService) Create(ctx context.Context, req dto.FaceCreateDTO) (*entity.FaceEmbedding, error) {
	// Check for duplicate student_id
	existing, err := s.repo.FindByStudentID(ctx, req.StudentID)
	if err == nil && existing != nil {
		return nil, ErrDuplicateStudentID
	}

	face := &entity.FaceEmbedding{
		Name:       req.Name,
		StudentID:  req.StudentID,
		RoomNumber: req.RoomNumber,
	}

	id, err := s.repo.Create(ctx, face)
	if err != nil {
		s.logger.Error("Failed to create face record", zap.Error(err))
		return nil, err
	}

	created, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// Update modifies an existing face record.
func (s *FaceService) Update(ctx context.Context, id int64, req dto.FaceUpdateDTO) (*entity.FaceEmbedding, error) {
	face, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrFaceNotFound
	}

	if req.Name != "" {
		face.Name = req.Name
	}
	if req.RoomNumber != "" {
		face.RoomNumber = req.RoomNumber
	}

	if err := s.repo.Update(ctx, face); err != nil {
		s.logger.Error("Failed to update face record", zap.Error(err))
		return nil, err
	}

	return face, nil
}

// Delete removes a face record by ID.
func (s *FaceService) Delete(ctx context.Context, id int64) error {
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return ErrFaceNotFound
	}
	return s.repo.Delete(ctx, id)
}

// List retrieves face records with pagination.
func (s *FaceService) List(ctx context.Context, page, size int) ([]entity.FaceEmbedding, int64, error) {
	return s.repo.FindAllPaginated(ctx, page, size)
}

func (s *FaceService) Enroll(ctx context.Context, req dto.FaceEnrollDTO) (*entity.FaceEmbedding, error) {
	existing, err := s.repo.FindByStudentID(ctx, req.StudentID)
	if err == nil && existing != nil {
		return nil, ErrDuplicateStudentID
	}

	now := time.Now()

	face := &entity.FaceEmbedding{
		Name:       req.Name,
		StudentID:  req.StudentID,
		RoomNumber: req.RoomNumber,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	embedding, extrErr := s.extractEmbedding(ctx, req.PhotoBytes)
	if extrErr != nil {
		s.logger.Warn("face_embedding_extraction_failed", zap.Error(extrErr))
	} else if len(embedding) > 0 {
		face.Embedding = floatsToBytes(embedding)
	}

	if len(req.PhotoBytes) > 0 {
		_ = os.MkdirAll(s.uploadDir, 0o755)
		filename := fmt.Sprintf("%s_%d.jpg", req.StudentID, now.UnixMilli())
		savePath := filepath.Join(s.uploadDir, filename)
		if err := os.WriteFile(savePath, req.PhotoBytes, 0o644); err != nil {
			s.logger.Warn("face_photo_save_failed", zap.Error(err))
		} else {
			face.ImagePath = jsontype.NullString{sql.NullString{String: savePath, Valid: true}}
		}
	}

	id, err := s.repo.Create(ctx, face)
	if err != nil {
		s.logger.Error("failed to create face record", zap.Error(err))
		return nil, err
	}

	created, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *FaceService) extractEmbedding(ctx context.Context, imageData []byte) ([]float32, error) {
	if s.faceRecognitionURL == "" || len(imageData) == 0 {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.faceRecognitionURL, bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", strconv.Itoa(len(imageData)))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("face-recognition api call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Success      bool       `json:"success"`
		Embedding    []float32  `json:"embedding,omitempty"`
		FaceDetected bool       `json:"face_detected"`
		Error        string     `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("face extraction failed: %s", result.Error)
	}

	if !result.FaceDetected || len(result.Embedding) == 0 {
		return nil, nil
	}

	return result.Embedding, nil
}

// floatsToBytes encodes a []float32 as little-endian 32-bit BLOB bytes.
func floatsToBytes(floats []float32) []byte {
	data := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := math.Float32bits(f)
		data[i*4+0] = byte(bits)
		data[i*4+1] = byte(bits >> 8)
		data[i*4+2] = byte(bits >> 16)
		data[i*4+3] = byte(bits >> 24)
	}
	return data
}

// BatchImport creates multiple face records.
func (s *FaceService) BatchImport(ctx context.Context, req dto.FaceBatchImportDTO) (int64, int64, error) {
	var created, duplicates int64
	for _, student := range req.Students {
		existing, err := s.repo.FindByStudentID(ctx, student.StudentID)
		if err == nil && existing != nil {
			duplicates++
			continue
		}

		face := &entity.FaceEmbedding{
			Name:       student.Name,
			StudentID:  student.StudentID,
			RoomNumber: student.RoomNumber,
		}

		_, err = s.repo.Create(ctx, face)
		if err != nil {
			s.logger.Error("Failed to batch import face record",
				zap.String("student_id", student.StudentID),
				zap.Error(err))
			continue
		}
		created++
	}
	return created, duplicates, nil
}
