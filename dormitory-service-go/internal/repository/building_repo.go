package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// BuildingRepository handles dorm_building table operations.
type BuildingRepository struct {
	*BaseRepository[entity.DormBuilding]
}

// NewBuildingRepository creates a new BuildingRepository.
func NewBuildingRepository(db *sqlx.DB) *BuildingRepository {
	return &BuildingRepository{
		BaseRepository: NewBaseRepository[entity.DormBuilding](db, "dorm_building"),
	}
}

// FindByCode finds a building by its code (A/B/C/D).
func (r *BuildingRepository) FindByCode(ctx context.Context, code string) (*entity.DormBuilding, error) {
	var b entity.DormBuilding
	query := "SELECT * FROM dorm_building WHERE code = ? LIMIT 1"
	err := r.DB.GetContext(ctx, &b, query, code)
	if err != nil {
		return nil, fmt.Errorf("building not found for code %s: %w", code, err)
	}
	return &b, nil
}
