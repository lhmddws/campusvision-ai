package entity

import (
	"database/sql"
	"time"
)

// DormCamera maps to the dorm_camera table.
// This is the camera information table storing RTSP stream configuration and status.
type DormCamera struct {
	ID              int64          `db:"id" json:"id"`
	CameraID        string         `db:"camera_id" json:"camera_id"`
	Name            string         `db:"name" json:"name"`
	Building        string         `db:"building" json:"building"`
	RtspURL         string         `db:"rtsp_url" json:"rtsp_url"`
	Direction       string         `db:"direction" json:"direction"`
	Resolution      string         `db:"resolution" json:"resolution"`
	Status          string         `db:"status" json:"status"`
	FPSCurrent      sql.NullFloat64 `db:"fps_current" json:"fps_current"`
	TotalFrames     sql.NullInt64  `db:"total_frames" json:"total_frames"`
	LastHeartbeat   sql.NullTime   `db:"last_heartbeat" json:"last_heartbeat"`
	LastEventTime   sql.NullTime   `db:"last_event_time" json:"last_event_time"`
	Enabled         bool           `db:"enabled" json:"enabled"`
	ConfigJSON      sql.NullString `db:"config_json" json:"config_json"`
	Remark          sql.NullString `db:"remark" json:"remark"`
	LastHealthCheck sql.NullTime   `db:"last_health_check" json:"last_health_check"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at" json:"updated_at"`
}
