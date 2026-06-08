package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// ConfigRepository handles dorm_config table operations.
type ConfigRepository struct {
	*BaseRepository[entity.DormConfig]
}

// NewConfigRepository creates a new ConfigRepository.
func NewConfigRepository(db *sqlx.DB) *ConfigRepository {
	return &ConfigRepository{
		BaseRepository: NewBaseRepository[entity.DormConfig](db, "dorm_config"),
	}
}

// FindByKey finds a configuration entry by its key.
func (r *ConfigRepository) FindByKey(ctx context.Context, configKey string) (*entity.DormConfig, error) {
	var cfg entity.DormConfig
	query := "SELECT * FROM dorm_config WHERE config_key = ? LIMIT 1"
	err := r.DB.GetContext(ctx, &cfg, query, configKey)
	if err != nil {
		return nil, fmt.Errorf("find config by key %s: %w", configKey, err)
	}
	return &cfg, nil
}

// FindByGroup finds all configuration entries in a group.
func (r *ConfigRepository) FindByGroup(ctx context.Context, groupName string) ([]entity.DormConfig, error) {
	var configs []entity.DormConfig
	query := "SELECT * FROM dorm_config WHERE group_name = ? ORDER BY config_key"
	err := r.DB.SelectContext(ctx, &configs, query, groupName)
	if err != nil {
		return nil, fmt.Errorf("find configs by group %s: %w", groupName, err)
	}
	return configs, nil
}

// UpdateByKey updates a configuration value by its key.
// Key existence must be validated by the caller (service layer does this).
func (r *ConfigRepository) UpdateByKey(ctx context.Context, configKey, configValue string) error {
	query := "UPDATE dorm_config SET config_value = ? WHERE config_key = ?"
	_, err := r.DB.ExecContext(ctx, query, configValue, configKey)
	if err != nil {
		return fmt.Errorf("update config %s: %w", configKey, err)
	}
	return nil
}

// BatchUpdateByKey updates multiple config values.
// Keys are trusted to exist (loaded from API by frontend).
func (r *ConfigRepository) BatchUpdateByKey(ctx context.Context, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}

	query := "UPDATE dorm_config SET config_value = ? WHERE config_key = ?"
	for key, value := range updates {
		if _, err := r.DB.ExecContext(ctx, query, value, key); err != nil {
			return fmt.Errorf("update config %s: %w", key, err)
		}
	}

	return nil
}
