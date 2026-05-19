package entity

import (
	"database/sql"
	"time"
)

// DormStudentStatus maps to the dorm_student_status table (init.sql schema).
// Tracks whether each student is currently in the dormitory.
type DormStudentStatus struct {
	ID              int64          `db:"id" json:"id"`
	StudentID       string         `db:"student_id" json:"student_id"`
	StudentName     string         `db:"student_name" json:"student_name"`
	Building        string         `db:"building" json:"building"`
	Room            sql.NullString `db:"room" json:"room"`
	IsInDorm        bool           `db:"is_in_dorm" json:"is_in_dorm"`
	LastEntryTime   sql.NullTime   `db:"last_entry_time" json:"last_entry_time"`
	LastExitTime    sql.NullTime   `db:"last_exit_time" json:"last_exit_time"`
	TodayStatus     string         `db:"today_status" json:"today_status"`
	TodayEntryCount int            `db:"today_entry_count" json:"today_entry_count"`
	TodayExitCount  int            `db:"today_exit_count" json:"today_exit_count"`
	LastUpdate      time.Time      `db:"last_update" json:"last_update"`
}
