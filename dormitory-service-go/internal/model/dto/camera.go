package dto

import (
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
)

// CameraCreateDTO is the request body for creating a new camera.
type CameraCreateDTO struct {
	CameraID   string `json:"camera_id" binding:"required"`
	Building   string `json:"building" binding:"required"`
	Name       string `json:"name" binding:"required"`
	RtspURL    string `json:"rtsp_url" binding:"required"`
	Direction  string `json:"direction"`
	Resolution string `json:"resolution"`
	Remark     string `json:"remark"`
	Type       string `json:"type"`
	Protocol   string `json:"protocol"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Path       string `json:"path"`
	Username   string `json:"username"`
}

// CameraUpdateDTO is the request body for updating an existing camera.
type CameraUpdateDTO struct {
	Name       string `json:"name"`
	Building   string `json:"building"`
	RtspURL    string `json:"rtsp_url"`
	Direction  string `json:"direction"`
	Resolution string `json:"resolution"`
	Enabled    *bool  `json:"enabled"`
	Status     string `json:"status"`
	Remark     string `json:"remark"`
	Type       string `json:"type"`
	Protocol   string `json:"protocol"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Path       string `json:"path"`
	Username   string `json:"username"`
}

// CameraResponse is the JSON response DTO for a camera, excluding sensitive fields
// (password_enc, nonce, key_id) that are used only internally.
type CameraResponse struct {
	ID              int64                `json:"id"`
	CameraID        string               `json:"camera_id"`
	Name            string               `json:"name"`
	Building        string               `json:"building"`
	RtspURL         string               `json:"rtsp_url"`             // Sanitized (password masked by ToDTO)
	SanitizedRtspURL string              `json:"sanitized_rtsp_url"`   // Always-masked RTSP URL
	Direction       string               `json:"direction"`
	Resolution      string               `json:"resolution"`
	Status          string               `json:"status"`
	FPSCurrent      jsontype.NullFloat64 `json:"fps_current"`
	TotalFrames     jsontype.NullInt64   `json:"total_frames"`
	LastHeartbeat   jsontype.NullTime    `json:"last_heartbeat"`
	LastEventTime   jsontype.NullTime    `json:"last_event_time"`
	Enabled         bool                 `json:"enabled"`
	ConfigJSON      jsontype.NullString  `json:"config_json"`
	Remark          jsontype.NullString  `json:"remark"`
	LastHealthCheck jsontype.NullTime    `json:"last_health_check"`
	Type            string               `json:"type"`
	Protocol        string               `json:"protocol"`
	Host            jsontype.NullString  `json:"host"`
	Port            jsontype.NullInt64   `json:"port"`
	Path            jsontype.NullString  `json:"path"`
	Username        jsontype.NullString  `json:"username"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}
