package enums

// EventType represents dormitory entry/exit event types.
type EventType string

const (
	EventTypeEntry EventType = "entry"
	EventTypeExit  EventType = "exit"
)

// IsValid returns true if the event type is recognized.
func (e EventType) IsValid() bool {
	switch e {
	case EventTypeEntry, EventTypeExit:
		return true
	default:
		return false
	}
}

// CameraStatus represents camera operational status.
type CameraStatus string

const (
	CameraStatusOnline  CameraStatus = "online"
	CameraStatusOffline CameraStatus = "offline"
	CameraStatusIdle    CameraStatus = "idle"
	CameraStatusUnknown CameraStatus = "unknown"
)

// AlertType represents alert category.
type AlertType string

const (
	AlertTypeStrangerEntry   AlertType = "STRANGER_ENTRY"
	AlertTypeLongAbsence     AlertType = "LONG_ABSENCE"
	AlertTypeCameraOffline   AlertType = "CAMERA_OFFLINE"
	AlertTypeCrossBuilding   AlertType = "CROSS_BUILDING"
	AlertTypeAbnormalBehavior AlertType = "ABNORMAL_BEHAVIOR"
	AlertTypeLateReturn      AlertType = "LATE_RETURN"
	AlertTypeSystem          AlertType = "SYSTEM"
)

// AttendanceStatus represents attendance status.
type AttendanceStatus string

const (
	AttendanceStatusPresent  AttendanceStatus = "PRESENT"
	AttendanceStatusAbsent   AttendanceStatus = "ABSENT"
	AttendanceStatusLate     AttendanceStatus = "LATE"
	AttendanceStatusStranger AttendanceStatus = "STRANGER"
	AttendanceStatusUnknown  AttendanceStatus = "unknown"
)

// StudentStatus represents whether a student is in the dorm.
type StudentStatus string

const (
	StudentStatusIn  StudentStatus = "in"
	StudentStatusOut StudentStatus = "out"
)

// GenderType represents gender.
type GenderType string

const (
	GenderTypeMale   GenderType = "MALE"
	GenderTypeFemale GenderType = "FEMALE"
)

// SeverityLevel represents alert severity.
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// StrangerStatus represents stranger record status.
type StrangerStatus string

const (
	StrangerStatusUnconfirmed StrangerStatus = "UNCONFIRMED"
	StrangerStatusConfirmed   StrangerStatus = "CONFIRMED"
	StrangerStatusDismissed   StrangerStatus = "DISMISSED"
)

// ReportStatus represents nightly report generation status.
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "PENDING"
	ReportStatusCompleted ReportStatus = "COMPLETED"
	ReportStatusFailed    ReportStatus = "FAILED"
)

// TriggerType represents how a nightly report was triggered.
type TriggerType string

const (
	TriggerTypeAuto   TriggerType = "AUTO"
	TriggerTypeManual TriggerType = "MANUAL"
)

// SyncType represents sync log type.
type SyncType string

const (
	SyncTypeStudent SyncType = "STUDENT"
)

// SyncStatus represents sync operation status.
type SyncStatus string

const (
	SyncStatusSuccess    SyncStatus = "SUCCESS"
	SyncStatusFailed     SyncStatus = "FAILED"
	SyncStatusInProgress SyncStatus = "IN_PROGRESS"
)

// NightlyDetailStatus represents student status in nightly detail.
type NightlyDetailStatus string

const (
	NightlyDetailPresent    NightlyDetailStatus = "present"
	NightlyDetailAbsent     NightlyDetailStatus = "absent"
	NightlyDetailLateReturn NightlyDetailStatus = "late_return"
	NightlyDetailUnknown    NightlyDetailStatus = "unknown"
)
