package entity

import (
	"database/sql"
	"time"
)

// DormSyncLog maps to the dorm_sync_log table (init.sql schema).
// Records data synchronization operations (e.g., student data sync).
type DormSyncLog struct {
	ID           int64          `db:"id" json:"id"`
	SyncType     string         `db:"sync_type" json:"sync_type"`
	SyncStatus   string         `db:"sync_status" json:"sync_status"`
	TotalCount   sql.NullInt64  `db:"total_count" json:"total_count"`
	SuccessCount sql.NullInt64  `db:"success_count" json:"success_count"`
	FailCount    sql.NullInt64  `db:"fail_count" json:"fail_count"`
	ErrorMessage sql.NullString `db:"error_message" json:"error_message"`
	DurationMs   sql.NullInt64  `db:"duration_ms" json:"duration_ms"`
	StartedAt    time.Time      `db:"started_at" json:"started_at"`
	FinishedAt   sql.NullTime   `db:"finished_at" json:"finished_at"`
}
