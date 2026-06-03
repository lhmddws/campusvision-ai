package service

import (
	"context"
	"errors"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"go.uber.org/zap"
)

var ErrFaceNotFound = errors.New("face record not found")
var ErrDuplicateStudentID = errors.New("student_id already exists")

// FaceService handles face record CRUD operations.
type FaceService struct {
	repo   *repository.FaceRepository
	logger *zap.Logger
}

// NewFaceService creates a new FaceService.
func NewFaceService(repo *repository.FaceRepository, logger *zap.Logger) *FaceService {
	return &FaceService{
		repo:   repo,
		logger: logger,
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