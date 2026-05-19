package entity

import (
	"database/sql"
	"time"
)

// DormStrangerRecord maps to the dorm_stranger_record table.
// Records detected strangers who are not matched to any known student.
type DormStrangerRecord struct {
	ID              int64          `db:"id" json:"id"`
	Building        string         `db:"building" json:"building"`
	FaceSnapshotURL sql.NullString `db:"face_snapshot_url" json:"face_snapshot_url"`
	Confidence      sql.NullFloat64 `db:"confidence" json:"confidence"`
	EventType       string         `db:"event_type" json:"event_type"`
	DetectedTime    time.Time      `db:"detected_time" json:"detected_time"`
	Status          string         `db:"status" json:"status"`
	Remark          sql.NullString `db:"remark" json:"remark"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
}
