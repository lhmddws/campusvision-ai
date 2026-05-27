package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// StudentRepository handles dorm_student table operations.
type StudentRepository struct {
	*BaseRepository[entity.DormStudent]
}

// NewStudentRepository creates a new StudentRepository.
func NewStudentRepository(db *sqlx.DB) *StudentRepository {
	return &StudentRepository{
		BaseRepository: NewBaseRepository[entity.DormStudent](db, "dorm_student_assignment"),
	}
}

// FindByStudentID finds a student by their student_id (学号).
func (r *StudentRepository) FindByStudentID(ctx context.Context, studentID string) (*entity.DormStudent, error) {
	var s entity.DormStudent
	query := "SELECT * FROM dorm_student_assignment WHERE student_id = ? LIMIT 1"
	err := r.DB.GetContext(ctx, &s, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("find student by id %s: %w", studentID, err)
	}
	return &s, nil
}

// FindByBuilding finds all students in a given building.
func (r *StudentRepository) FindByBuilding(ctx context.Context, building string) ([]entity.DormStudent, error) {
	var students []entity.DormStudent
	query := "SELECT * FROM dorm_student_assignment WHERE building = ? ORDER BY student_id"
	err := r.DB.SelectContext(ctx, &students, query, building)
	if err != nil {
		return nil, fmt.Errorf("find students by building %s: %w", building, err)
	}
	return students, nil
}

// FindByRoom finds all students in a given room.
func (r *StudentRepository) FindByRoom(ctx context.Context, room string) ([]entity.DormStudent, error) {
	var students []entity.DormStudent
	query := "SELECT * FROM dorm_student_assignment WHERE room = ? ORDER BY student_id"
	err := r.DB.SelectContext(ctx, &students, query, room)
	if err != nil {
		return nil, fmt.Errorf("find students by room %s: %w", room, err)
	}
	return students, nil
}

// FindWithPagination paginates students with optional building filter.
func (r *StudentRepository) FindWithPagination(ctx context.Context, building string, page, size int) ([]entity.DormStudent, int64, error) {
	where := ""
	var args []interface{}
	if building != "" {
		where = "building = ?"
		args = append(args, building)
	}
	return r.BaseRepository.FindWithPagination(ctx, where, args, "student_id ASC", page, size)
}
