package entity

import (
	"database/sql"
	"time"
)

// DormStudentAssignment maps to the dorm_student_assignment table (init.sql schema).
// This is the student dormitory assignment record synced from the student management system.
type DormStudentAssignment struct {
	ID          int64          `db:"id" json:"id"`
	StudentID   string         `db:"student_id" json:"student_id"`
	StudentName string         `db:"student_name" json:"student_name"`
	Building    string         `db:"building" json:"building"`
	Room        string         `db:"room" json:"room"`
	ClassName   sql.NullString `db:"class_name" json:"class_name"`
	Grade       sql.NullString `db:"grade" json:"grade"`
	Gender      sql.NullString `db:"gender" json:"gender"`
	Phone       sql.NullString `db:"phone" json:"phone"`
	Active      bool           `db:"active" json:"active"`
	SyncVersion int64          `db:"sync_version" json:"sync_version"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
}
