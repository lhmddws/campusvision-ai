package repository

import (
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
		BaseRepository: NewBaseRepository[entity.DormStudent](db, "dorm_student"),
	}
}

// FindByStudentID finds a student by their student_id (学号).
func (r *StudentRepository) FindByStudentID(studentID string) (*entity.DormStudent, error) {
	var s entity.DormStudent
	query := "SELECT * FROM dorm_student WHERE student_id = ? LIMIT 1"
	err := r.DB.Get(&s, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("find student by id %s: %w", studentID, err)
	}
	return &s, nil
}

// FindByBuilding finds all students in a given building.
func (r *StudentRepository) FindByBuilding(buildingID int64) ([]entity.DormStudent, error) {
	var students []entity.DormStudent
	query := "SELECT * FROM dorm_student WHERE building_id = ? ORDER BY student_id"
	err := r.DB.Select(&students, query, buildingID)
	if err != nil {
		return nil, fmt.Errorf("find students by building %d: %w", buildingID, err)
	}
	return students, nil
}

// FindByRoom finds all students in a given room.
func (r *StudentRepository) FindByRoom(roomID int64) ([]entity.DormStudent, error) {
	var students []entity.DormStudent
	query := "SELECT * FROM dorm_student WHERE room_id = ? ORDER BY student_id"
	err := r.DB.Select(&students, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("find students by room %d: %w", roomID, err)
	}
	return students, nil
}

// FindByStatus finds all students with a given status.
func (r *StudentRepository) FindByStatus(status string) ([]entity.DormStudent, error) {
	var students []entity.DormStudent
	query := "SELECT * FROM dorm_student WHERE status = ? ORDER BY student_id"
	err := r.DB.Select(&students, query, status)
	if err != nil {
		return nil, fmt.Errorf("find students by status %s: %w", status, err)
	}
	return students, nil
}

// FindWithPagination paginates students with optional building filter.
func (r *StudentRepository) FindWithPagination(buildingID int64, page, size int) ([]entity.DormStudent, int64, error) {
	where := ""
	var args []interface{}
	if buildingID > 0 {
		where = "building_id = ?"
		args = append(args, buildingID)
	}
	return r.BaseRepository.FindWithPagination(where, args, "student_id ASC", page, size)
}
