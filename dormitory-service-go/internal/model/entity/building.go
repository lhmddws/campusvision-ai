package entity

import (
	"database/sql"
	"time"
)

// DormBuilding maps to the dorm_building table.
// Represents a dormitory building. Note: this table does not exist in init.sql;
// buildings (A/B/C/D) are typically resolved from code or Redis cache.
type DormBuilding struct {
	ID            int64          `db:"id" json:"id"`
	Name          string         `db:"name" json:"name"`
	Code          string         `db:"code" json:"code"`
	Floors        sql.NullInt64  `db:"floors" json:"floors"`
	RoomsPerFloor sql.NullInt64  `db:"rooms_per_floor" json:"rooms_per_floor"`
	Description   sql.NullString `db:"description" json:"description"`
	Enabled       bool           `db:"enabled" json:"enabled"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
}
