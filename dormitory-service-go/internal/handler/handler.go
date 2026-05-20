package handler

import (
	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

// Handler aggregates all service dependencies for HTTP handlers.
type Handler struct {
	CameraService *service.CameraService
	RecordService *service.RecordService
	AlertService  *service.AlertService
	ConfigService *service.ConfigService
	ReportService *service.ReportService
	DB            *sqlx.DB
}

// NewHandler creates a Handler with all required services.
func NewHandler(
	cameraService *service.CameraService,
	recordService *service.RecordService,
	alertService *service.AlertService,
	configService *service.ConfigService,
	reportService *service.ReportService,
	db *sqlx.DB,
) *Handler {
	return &Handler{
		CameraService: cameraService,
		RecordService: recordService,
		AlertService:  alertService,
		ConfigService: configService,
		ReportService: reportService,
		DB:            db,
	}
}
