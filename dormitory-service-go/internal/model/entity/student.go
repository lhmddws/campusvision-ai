package entity

import (
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
)

// DormStudent maps to the dorm_student_assignment table.
// Represents a student's dormitory assignment.
type DormStudent struct {
	ID          int64               `db:"id" json:"id"`
	StudentID   string              `db:"student_id" json:"student_id"`
	StudentName string              `db:"student_name" json:"student_name"`
	Building    string              `db:"building" json:"building"`
	Room        string              `db:"room" json:"room"`
	ClassName   jsontype.NullString `db:"class_name" json:"class_name"`
	Grade       jsontype.NullString `db:"grade" json:"grade"`
	Gender      jsontype.NullString `db:"gender" json:"gender"`
	Phone       jsontype.NullString `db:"phone" json:"phone"`
	Active      bool                `db:"active" json:"active"`
	SyncVersion int64               `db:"sync_version" json:"sync_version"`
	CreatedAt   time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time           `db:"updated_at" json:"updated_at"`
}
