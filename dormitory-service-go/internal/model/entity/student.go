package entity

import (
	"database/sql"
	"time"
)

// DormStudent maps to the dorm_student table.
// Represents a student's dormitory assignment and status.
// Note: init.sql names this table dorm_student_assignment; the Go service uses
// dorm_student for compatibility with the existing Java entity convention.
type DormStudent struct {
	ID            int64          `db:"id" json:"id"`
	StudentID     string         `db:"student_id" json:"student_id"`
	Name          string         `db:"name" json:"name"`
	Gender        sql.NullString `db:"gender" json:"gender"`
	BuildingID    sql.NullInt64  `db:"building_id" json:"building_id"`
	RoomID        sql.NullInt64  `db:"room_id" json:"room_id"`
	BedNumber     sql.NullString `db:"bed_number" json:"bed_number"`
	Status        sql.NullString `db:"status" json:"status"`
	LastEventTime sql.NullTime   `db:"last_event_time" json:"last_event_time"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
}
