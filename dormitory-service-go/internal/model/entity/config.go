package entity

import (
	"database/sql"
	"time"
)

// DormConfig maps to the dorm_config table.
// Stores key-value configuration entries for the system.
type DormConfig struct {
	ID           int64          `db:"id" json:"id"`
	ConfigKey    string         `db:"config_key" json:"config_key"`
	ConfigValue  string         `db:"config_value" json:"config_value"`
	ConfigType   sql.NullString `db:"config_type" json:"config_type"`
	Description  sql.NullString `db:"description" json:"description"`
	DefaultValue sql.NullString `db:"default_value" json:"default_value"`
	GroupName    sql.NullString `db:"group_name" json:"group_name"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
}
