package entity

import (
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
)

// DormAlert maps to the dorm_alert_record table.
// Stores alert records triggered by various conditions (stranger, absence, etc.).
type DormAlert struct {
	ID              int64               `db:"id" json:"id"`
	AlertID         string              `db:"alert_id" json:"alert_id"`
	AlertType       string              `db:"alert_type" json:"alert_type"`
	Building        jsontype.NullString `db:"building" json:"building"`
	StudentID       jsontype.NullString `db:"student_id" json:"student_id"`
	Severity        string              `db:"severity" json:"severity"`
	Description     jsontype.NullString `db:"description" json:"description"`
	FaceSnapshotURL jsontype.NullString `db:"face_snapshot_url" json:"face_snapshot_url"`
	IsRead          bool                `db:"is_read" json:"is_read"`
	IsResolved      bool                `db:"is_resolved" json:"is_resolved"`
	OccurredAt      time.Time           `db:"occurred_at" json:"occurred_at"`
	CreatedAt       time.Time           `db:"created_at" json:"created_at"`
}
