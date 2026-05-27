package entity

import (
	"database/sql"
	"time"
)

// DormEventLog maps to the dorm_entry_exit_event table.
// This is the core event stream recording entry/exit events from face recognition.
// All 13 columns align with infra/mariadb/init.sql's dorm_entry_exit_event table.
// Building is stored as a string code (A/B/C/D) matching init.sql's building VARCHAR(8).
type DormEventLog struct {
	ID             int64          `db:"id" json:"id"`
	EventID        string         `db:"event_id" json:"event_id"`
	CameraID       sql.NullString `db:"camera_id" json:"camera_id"`
	Building       string         `db:"building" json:"building"`
	EventType      string         `db:"event_type" json:"event_type"`
	StudentID      sql.NullString `db:"student_id" json:"student_id"`
	StudentName    sql.NullString `db:"student_name" json:"student_name"`
	IsStranger     bool           `db:"is_stranger" json:"is_stranger"`
	IsProcessed    bool           `db:"is_processed" json:"is_processed"`
	Confidence     sql.NullFloat64 `db:"confidence" json:"confidence"`
	FaceSnapshotURL sql.NullString `db:"face_snapshot_url" json:"face_snapshot_url"`
	Timestamp      time.Time      `db:"timestamp" json:"timestamp"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
}
