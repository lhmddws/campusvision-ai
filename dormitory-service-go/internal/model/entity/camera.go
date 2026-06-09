package entity

import (
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
)

// DormCamera maps to the dorm_camera table.
// This is the camera information table storing RTSP stream configuration and status.
type DormCamera struct {
	ID              int64                `db:"id" json:"id"`
	CameraID        string               `db:"camera_id" json:"camera_id"`
	Name            string               `db:"name" json:"name"`
	Building        string               `db:"building" json:"building"`
	RtspURL         string               `db:"rtsp_url" json:"rtsp_url"`
	Direction       string               `db:"direction" json:"direction"`
	Resolution      string               `db:"resolution" json:"resolution"`
	Status          string               `db:"status" json:"status"`
	FPSCurrent      jsontype.NullFloat64 `db:"fps_current" json:"fps_current"`
	TotalFrames     jsontype.NullInt64   `db:"total_frames" json:"total_frames"`
	LastHeartbeat   jsontype.NullTime    `db:"last_heartbeat" json:"last_heartbeat"`
	LastEventTime   jsontype.NullTime    `db:"last_event_time" json:"last_event_time"`
	Enabled         bool                 `db:"enabled" json:"enabled"`
	ConfigJSON      jsontype.NullString  `db:"config_json" json:"config_json"`
	Remark          jsontype.NullString  `db:"remark" json:"remark"`
	LastHealthCheck jsontype.NullTime    `db:"last_health_check" json:"last_health_check"`
	PasswordEnc     jsontype.NullString  `db:"password_enc" json:"password_enc"`
	Nonce           jsontype.NullString  `db:"nonce" json:"nonce"`
	Type            string               `db:"type" json:"type"`
	Protocol        string               `db:"protocol" json:"protocol"`
	Host            jsontype.NullString  `db:"host" json:"host"`
	Port            jsontype.NullInt64   `db:"port" json:"port"`
	Path            jsontype.NullString  `db:"path" json:"path"`
	Username        jsontype.NullString  `db:"username" json:"username"`
	KeyID           jsontype.NullString  `db:"key_id" json:"key_id"`
	CreatedAt       time.Time            `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time            `db:"updated_at" json:"updated_at"`
}
