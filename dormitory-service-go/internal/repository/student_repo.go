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

// InspectionRow represents a single row from the inspection list query.
type InspectionRow struct {
	Building     string `db:"building"`
	Room         string `db:"room"`
	StudentID    string `db:"student_id"`
	StudentName  string `db:"student_name"`
	TodayStatus  string `db:"today_status"`
}

// FindInspectionList returns all active students in a building with their today's
// latest entry/exit status, computed via a correlated subquery on dorm_entry_exit_event.
func (r *StudentRepository) FindInspectionList(ctx context.Context, building string, today string) ([]InspectionRow, error) {
	query := `
		SELECT s.building, s.room, s.student_id, s.student_name,
		       COALESCE(
		         (SELECT e.event_type FROM dorm_entry_exit_event e
		          WHERE e.student_id = s.student_id
		            AND DATE(e.timestamp) = ?
		          ORDER BY e.timestamp DESC LIMIT 1
		         ), 'unknown'
		       ) as today_status
		FROM dorm_student_assignment s
		WHERE s.building = ? AND s.active = 1
		ORDER BY s.room, s.student_name
	`
	var rows []InspectionRow
	err := r.DB.SelectContext(ctx, &rows, query, today, building)
	if err != nil {
		return nil, fmt.Errorf("find inspection list for building %s: %w", building, err)
	}
	return rows, nil
}

// CountActiveByBuilding returns the number of active students in a building.
func (r *StudentRepository) CountActiveByBuilding(ctx context.Context, building string) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM dorm_student_assignment WHERE building = ? AND active = 1"
	err := r.DB.GetContext(ctx, &count, query, building)
	if err != nil {
		return 0, fmt.Errorf("count active students by building %s: %w", building, err)
	}
	return count, nil
}

// CountActiveAll returns the number of active students across all buildings.
func (r *StudentRepository) CountActiveAll(ctx context.Context) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM dorm_student_assignment WHERE active = 1"
	err := r.DB.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("count active students all: %w", err)
	}
	return count, nil
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
