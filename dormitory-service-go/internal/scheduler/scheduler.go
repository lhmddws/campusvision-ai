package scheduler

import (
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Manager manages all scheduled cron jobs.
type Manager struct {
	logger *zap.Logger
	cron   *cron.Cron
	jobs   []cron.EntryID
}

// NewManager creates a new scheduler Manager.
func NewManager(logger *zap.Logger) *Manager {
	c := cron.New(cron.WithLocation(timeZone()))
	return &Manager{
		logger: logger,
		cron:   c,
		jobs:   make([]cron.EntryID, 0),
	}
}

// AddJob registers a cron job with a schedule expression.
func (m *Manager) AddJob(schedule string, job cron.Job) {
	id, err := m.cron.AddJob(schedule, job)
	if err != nil {
		m.logger.Error("Failed to register cron job",
			zap.String("schedule", schedule),
			zap.Error(err),
		)
		return
	}
	m.jobs = append(m.jobs, id)
	m.logger.Info("Registered cron job",
		zap.String("schedule", schedule),
		zap.Int("entry_id", int(id)),
	)
}

// Start begins all scheduled jobs.
func (m *Manager) Start() {
	m.logger.Info("Starting scheduler",
		zap.Int("jobs", len(m.jobs)),
	)
	m.cron.Start()
}

// Stop gracefully stops all scheduled jobs.
func (m *Manager) Stop() {
	m.logger.Info("Stopping scheduler")
	ctx := m.cron.Stop()
	<-ctx.Done()
	m.logger.Info("Scheduler stopped")
}

// timeZone returns the Asia/Shanghai timezone for cron scheduling.
func timeZone() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// Fallback to UTC if timezone data is unavailable
		return time.UTC
	}
	return loc
}

// getTodayDate returns today's date in YYYY-MM-DD format in Asia/Shanghai.
func getTodayDate() string {
	return time.Now().In(timeZone()).Format("2006-01-02")
}
