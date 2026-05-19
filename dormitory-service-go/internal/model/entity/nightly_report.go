package entity

import (
	"database/sql"
	"time"
)

// DormNightlyReport maps to the dorm_nightly_report table.
// Stores nightly attendance summary statistics per building.
type DormNightlyReport struct {
	ID              int64        `db:"id" json:"id"`
	ReportDate      string       `db:"report_date" json:"report_date"`
	Building        string       `db:"building" json:"building"`
	TotalCount      int          `db:"total_count" json:"total_count"`
	PresentCount    int          `db:"present_count" json:"present_count"`
	AbsentCount     int          `db:"absent_count" json:"absent_count"`
	LateReturnCount int          `db:"late_return_count" json:"late_return_count"`
	StrangerCount   int          `db:"stranger_count" json:"stranger_count"`
	UnknownCount    int          `db:"unknown_count" json:"unknown_count"`
	Status          string       `db:"status" json:"status"`
	TriggerType     string       `db:"trigger_type" json:"trigger_type"`
	CreatedAt       time.Time    `db:"created_at" json:"created_at"`
}

// DormNightlyDetail maps to the dorm_nightly_detail table.
// Stores per-student breakdown within a nightly report.
type DormNightlyDetail struct {
	ID          int64          `db:"id" json:"id"`
	ReportID    int64          `db:"report_id" json:"report_id"`
	StudentID   string         `db:"student_id" json:"student_id"`
	StudentName string         `db:"student_name" json:"student_name"`
	Building    string         `db:"building" json:"building"`
	Room        sql.NullString `db:"room" json:"room"`
	ClassName   sql.NullString `db:"class_name" json:"class_name"`
	Status      string         `db:"status" json:"status"`
	EntryTime   sql.NullTime   `db:"entry_time" json:"entry_time"`
	ExitTime    sql.NullTime   `db:"exit_time" json:"exit_time"`
	IsLateReturn bool          `db:"is_late_return" json:"is_late_return"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
}
