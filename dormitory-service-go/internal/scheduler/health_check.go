package scheduler

import (
	"go.uber.org/zap"

	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

// HealthCheckJob performs periodic health checks on all enabled cameras.
// Runs every 5 minutes.
type HealthCheckJob struct {
	logger      *zap.Logger
	cameraSvc   *service.CameraService
}

// NewHealthCheckJob creates a new HealthCheckJob.
func NewHealthCheckJob(logger *zap.Logger, cameraSvc *service.CameraService) *HealthCheckJob {
	return &HealthCheckJob{
		logger:    logger,
		cameraSvc: cameraSvc,
	}
}

// Run executes health checks for all enabled cameras.
// Implements cron.Job interface.
func (j *HealthCheckJob) Run() {
	cameras, err := j.cameraSvc.ListEnabledCameras()
	if err != nil {
		j.logger.Error("Failed to list enabled cameras for health check",
			zap.Error(err),
		)
		return
	}

	if len(cameras) == 0 {
		j.logger.Debug("No enabled cameras to health check")
		return
	}

	j.logger.Info("Starting health check for cameras",
		zap.Int("count", len(cameras)),
	)

	for _, cam := range cameras {
		if err := j.cameraSvc.HealthCheck(cam.CameraID); err != nil {
			j.logger.Warn("Camera health check failed",
				zap.String("camera_id", cam.CameraID),
				zap.String("building", cam.Building),
				zap.Error(err),
			)
			continue
		}
		j.logger.Debug("Camera health check completed",
			zap.String("camera_id", cam.CameraID),
		)
	}

	j.logger.Info("Health check cycle completed",
		zap.Int("checked", len(cameras)),
	)
}
