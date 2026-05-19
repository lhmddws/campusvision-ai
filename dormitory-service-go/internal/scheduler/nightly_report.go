package scheduler

import (
	"time"

	"go.uber.org/zap"

	"github.com/sims/campusvision/dormitory-service-go/internal/service"
)

// buildings is the list of known dormitory building codes.
var buildings = []string{"A", "B", "C", "D"}

// NightlyReportJob generates nightly attendance reports for all buildings.
// Runs daily at 23:00.
type NightlyReportJob struct {
	logger       *zap.Logger
	reportSvc    *service.ReportService
}

// NewNightlyReportJob creates a new NightlyReportJob.
func NewNightlyReportJob(logger *zap.Logger, reportSvc *service.ReportService) *NightlyReportJob {
	return &NightlyReportJob{
		logger:    logger,
		reportSvc: reportSvc,
	}
}

// Run executes the nightly report generation for all buildings.
// Implements cron.Job interface.
func (j *NightlyReportJob) Run() {
	today := time.Now().Format("2006-01-02")
	j.logger.Info("Starting nightly report generation",
		zap.String("date", today),
		zap.Any("buildings", buildings),
	)

	for _, building := range buildings {
		report, err := j.reportSvc.GenerateNightlyReport(building, today)
		if err != nil {
			j.logger.Error("Failed to generate nightly report",
				zap.String("building", building),
				zap.String("date", today),
				zap.Error(err),
			)
			continue
		}
		j.logger.Info("Nightly report generated",
			zap.String("building", building),
			zap.String("date", today),
			zap.Int64("report_id", report.ID),
		)
	}

	j.logger.Info("Nightly report generation completed",
		zap.String("date", today),
	)
}
