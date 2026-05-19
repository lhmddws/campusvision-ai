package dto

import "time"

// NightlyReportDTO is the API response DTO for a nightly report.
type NightlyReportDTO struct {
	ID              int64     `json:"id"`
	Building        string    `json:"building"`
	ReportDate      string    `json:"report_date"`
	TotalCount      int       `json:"total_count"`
	PresentCount    int       `json:"present_count"`
	AbsentCount     int       `json:"absent_count"`
	LateReturnCount int       `json:"late_return_count"`
	StrangerCount   int       `json:"stranger_count"`
	UnknownCount    int       `json:"unknown_count"`
	Status          string    `json:"status"`
	TriggerType     string    `json:"trigger_type"`
	CreatedAt       time.Time `json:"created_at"`
}

// NightlyReportQueryDTO is the query parameters for filtering nightly reports.
type NightlyReportQueryDTO struct {
	Building  string `json:"building" form:"building"`
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`
	Page      int    `json:"page" form:"page"`
	Size      int    `json:"size" form:"size"`
}
