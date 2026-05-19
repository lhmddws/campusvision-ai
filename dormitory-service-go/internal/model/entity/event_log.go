package entity

import (
	"database/sql"
	"time"
)

// DormEventLog maps to the dorm_event_log table.
// This is the core event stream recording entry/exit events from face recognition.
// Note: init.sql names this table dorm_entry_exit_event; the Go service uses
// dorm_event_log for compatibility with the existing Java entity convention.
type DormEventLog struct {
	ID           int64          `db:"id" json:"id"`
	CameraID     sql.NullString `db:"camera_id" json:"camera_id"`
	BuildingID   sql.NullInt64  `db:"building_id" json:"building_id"`
	EventType    string         `db:"event_type" json:"event_type"`
	StudentID    sql.NullString `db:"student_id" json:"student_id"`
	IsStranger   bool           `db:"is_stranger" json:"is_stranger"`
	Confidence   sql.NullFloat64 `db:"confidence" json:"confidence"`
	SnapshotPath sql.NullString `db:"snapshot_path" json:"snapshot_path"`
	Timestamp    time.Time      `db:"timestamp" json:"timestamp"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
}
