package dto

import "time"

// AttendanceQueryDTO is the query parameters for filtering attendance records.
type AttendanceQueryDTO struct {
	BuildingID int64      `json:"building_id" form:"building_id"`
	RoomID     int64      `json:"room_id" form:"room_id"`
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
	Status     string     `json:"status" form:"status"`
	Page       int        `json:"page" form:"page"`
	Size       int        `json:"size" form:"size"`
}

// AttendanceStatsDTO is the API response DTO for attendance statistics.
type AttendanceStatsDTO struct {
	Total    int64   `json:"total"`
	Present  int64   `json:"present"`
	Absent   int64   `json:"absent"`
	Late     int64   `json:"late"`
	Stranger int64   `json:"stranger"`
	Rate     float64 `json:"rate"`
}

// DailySummaryDTO is the API response DTO for daily attendance summary.
type DailySummaryDTO struct {
	Date         string  `json:"date"`
	BuildingName string  `json:"building_name"`
	CheckinRate  float64 `json:"checkin_rate"`
}

// StudentStatusDTO is the DTO for a student's current status.
type StudentStatusDTO struct {
	BuildingID int64  `json:"building_id"`
	RoomID     int64  `json:"room_id"`
	Status     string `json:"status"`
}
