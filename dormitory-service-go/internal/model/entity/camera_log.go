package entity

import (
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
)

// DormCameraLog maps to the dorm_camera_log table.
// Records camera status change events (online/offline transitions).
type DormCameraLog struct {
	ID         int64                `db:"id" json:"id"`
	CameraID   string               `db:"camera_id" json:"camera_id"`
	Building   string               `db:"building" json:"building"`
	StatusFrom jsontype.NullString  `db:"status_from" json:"status_from"`
	StatusTo   string               `db:"status_to" json:"status_to"`
	Reason     jsontype.NullString  `db:"reason" json:"reason"`
	FPSAtTime  jsontype.NullFloat64 `db:"fps_at_time" json:"fps_at_time"`
	CreatedAt  time.Time            `db:"created_at" json:"created_at"`
}
