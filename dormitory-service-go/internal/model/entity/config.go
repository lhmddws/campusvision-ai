package entity

import (
	"time"
)

// DormConfig maps to the dorm_config table.
// Stores key-value configuration entries for the system.
// Uses *string for nullable fields so JSON marshals to plain string or null.
type DormConfig struct {
	ID           int64     `db:"id" json:"id"`
	ConfigKey    string    `db:"config_key" json:"config_key"`
	ConfigValue  string    `db:"config_value" json:"config_value"`
	ConfigType   *string   `db:"config_type" json:"config_type"`
	Description  *string   `db:"description" json:"description"`
	DefaultValue  *string   `db:"default_value" json:"default_value"`
	ConfigOptions *string   `db:"config_options" json:"config_options"`
	GroupName     *string   `db:"group_name" json:"group_name"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
