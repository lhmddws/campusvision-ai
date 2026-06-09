package dto

import (
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
)

// AlertDTO is the API response DTO for an alert record.
type AlertDTO struct {
	ID              int64               `json:"id"`
	AlertID         string              `json:"alert_id"`
	AlertType       string              `json:"alert_type"`
	Building        jsontype.NullString `json:"building"`
	StudentID       jsontype.NullString `json:"student_id"`
	Severity        string              `json:"severity"`
	Description     jsontype.NullString `json:"description"`
	FaceSnapshotURL jsontype.NullString `json:"face_snapshot_url"`
	IsRead          bool                `json:"is_read"`
	IsResolved      bool                `json:"is_resolved"`
	OccurredAt      time.Time           `json:"occurred_at"`
	CreatedAt       time.Time           `json:"created_at"`
}

// AlertQueryDTO is the query parameters for filtering alert records.
type AlertQueryDTO struct {
	BuildingID   int64      `json:"building_id" form:"building_id"`
	AlertType    string     `json:"alert_type" form:"alert_type"`
	Acknowledged *bool      `json:"acknowledged" form:"acknowledged"`
	StartDate    *time.Time `json:"start_date" form:"start_date"`
	EndDate      *time.Time `json:"end_date" form:"end_date"`
	Page         int        `json:"page" form:"page"`
	Size         int        `json:"size" form:"size"`
}
