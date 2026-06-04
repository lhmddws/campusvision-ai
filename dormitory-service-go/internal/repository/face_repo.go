package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// FaceRepository provides CRUD operations for face_embedding records.
type FaceRepository struct {
	*BaseRepository[entity.FaceEmbedding]
}

// NewFaceRepository creates a new FaceRepository.
func NewFaceRepository(db *sqlx.DB) *FaceRepository {
	return &FaceRepository{
		BaseRepository: NewBaseRepository[entity.FaceEmbedding](db, "face_embedding"),
	}
}

// FindByStudentID retrieves a face record by student_id.
func (r *FaceRepository) FindByStudentID(ctx context.Context, studentID string) (*entity.FaceEmbedding, error) {
	var face entity.FaceEmbedding
	query := "SELECT id, name, student_id, room_number, image_path, created_at, updated_at FROM face_embedding WHERE student_id = ? LIMIT 1"
	err := r.DB.GetContext(ctx, &face, query, studentID)
	if err != nil {
		return nil, err
	}
	return &face, nil
}

// FindAllPaginated retrieves face records with pagination, excluding embedding BLOB.
func (r *FaceRepository) FindAllPaginated(ctx context.Context, page, size int) ([]entity.FaceEmbedding, int64, error) {
	var total int64
	err := r.DB.GetContext(ctx, &total, "SELECT COUNT(*) FROM face_embedding")
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	var faces []entity.FaceEmbedding
	query := "SELECT id, name, student_id, room_number, image_path, created_at, updated_at FROM face_embedding ORDER BY id DESC LIMIT ? OFFSET ?"
	err = r.DB.SelectContext(ctx, &faces, query, size, offset)
	if err != nil {
		return nil, 0, err
	}

	return faces, total, nil
}