package entity

import "time"

// DormBuilding maps to the dorm_building table.
// Columns: id BIGINT AUTO_INCREMENT, code VARCHAR(8) UNIQUE, name VARCHAR(64), created_at DATETIME.
type DormBuilding struct {
	ID        int64     `db:"id" json:"id"`
	Code      string    `db:"code" json:"code"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
