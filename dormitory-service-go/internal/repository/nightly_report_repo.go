package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// NightlyReportRepository handles dorm_nightly_report table operations.
type NightlyReportRepository struct {
	*BaseRepository[entity.DormNightlyReport]
}

// NewNightlyReportRepository creates a new NightlyReportRepository.
func NewNightlyReportRepository(db *sqlx.DB) *NightlyReportRepository {
	return &NightlyReportRepository{
		BaseRepository: NewBaseRepository[entity.DormNightlyReport](db, "dorm_nightly_report"),
	}
}

// FindByBuilding finds reports for a building.
func (r *NightlyReportRepository) FindByBuilding(building string, limit int) ([]entity.DormNightlyReport, error) {
	if limit <= 0 {
		limit = 100
	}
	var reports []entity.DormNightlyReport
	query := "SELECT * FROM dorm_nightly_report WHERE building = ? ORDER BY report_date DESC LIMIT ?"
	err := r.DB.Select(&reports, query, building, limit)
	if err != nil {
		return nil, fmt.Errorf("find reports by building %s: %w", building, err)
	}
	return reports, nil
}

// FindByDate finds reports for a specific date.
func (r *NightlyReportRepository) FindByDate(date string) ([]entity.DormNightlyReport, error) {
	var reports []entity.DormNightlyReport
	query := "SELECT * FROM dorm_nightly_report WHERE report_date = ? ORDER BY building"
	err := r.DB.Select(&reports, query, date)
	if err != nil {
		return nil, fmt.Errorf("find reports by date %s: %w", date, err)
	}
	return reports, nil
}

// FindByBuildingAndDate finds a report for a specific building and date.
func (r *NightlyReportRepository) FindByBuildingAndDate(building, date string) (*entity.DormNightlyReport, error) {
	var report entity.DormNightlyReport
	query := "SELECT * FROM dorm_nightly_report WHERE building = ? AND report_date = ? LIMIT 1"
	err := r.DB.Get(&report, query, building, date)
	if err != nil {
		return nil, fmt.Errorf("find report by building %s and date %s: %w", building, date, err)
	}
	return &report, nil
}

// FindWithPagination paginates nightly reports.
func (r *NightlyReportRepository) FindWithPagination(
	building string,
	startDate, endDate string,
	page, size int,
) ([]entity.DormNightlyReport, int64, error) {
	where := ""
	var args []interface{}
	conditions := []string{}

	if building != "" {
		conditions = append(conditions, "building = ?")
		args = append(args, building)
	}
	if startDate != "" {
		conditions = append(conditions, "report_date >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		conditions = append(conditions, "report_date <= ?")
		args = append(args, endDate)
	}

	if len(conditions) > 0 {
		where = conditions[0]
		for i := 1; i < len(conditions); i++ {
			where += " AND " + conditions[i]
		}
	}

	return r.BaseRepository.FindWithPagination(where, args, "report_date DESC, building ASC", page, size)
}
