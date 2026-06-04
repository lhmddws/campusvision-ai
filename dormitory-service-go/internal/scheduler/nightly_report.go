package scheduler

import (
	"go.uber.org/zap"
)

// GenerateNightlyReport generates the daily attendance summary report.
// Runs daily at 23:00 Asia/Shanghai.
type GenerateNightlyReport struct {
	logger *zap.Logger
}

// NewGenerateNightlyReport creates a new GenerateNightlyReport job.
func NewGenerateNightlyReport(logger *zap.Logger) *GenerateNightlyReport {
	return &GenerateNightlyReport{
		logger: logger,
	}
}

// Run executes the nightly report generation.
// Implements cron.Job interface.
func (j *GenerateNightlyReport) Run() {
	j.logger.Info("Starting nightly report generation")

	// TODO: Implement nightly report generation logic.
	// Steps:
	// 1. Query dorm_entry_exit_event for today's records
	// 2. Aggregate by student/building
	// 3. Generate summary statistics (total entries, exits, anomalies)
	// 4. Persist report to dorm_nightly_report table
	// 5. Log completion or failure

	j.logger.Info("Nightly report generation completed (stub)")
}
