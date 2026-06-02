package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"go.uber.org/zap"
)

// ConfigService handles configuration CRUD operations.
type ConfigService struct {
	configRepo *repository.ConfigRepository
	logger     *zap.Logger
}

// NewConfigService creates a new ConfigService.
func NewConfigService(configRepo *repository.ConfigRepository, logger *zap.Logger) *ConfigService {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	return &ConfigService{
		configRepo: configRepo,
		logger:     logger,
	}
}

func (s *ConfigService) log() *zap.Logger {
	if s.logger == nil {
		return zap.NewNop()
	}
	return s.logger
}

// GetAllConfigs returns all configs, optionally filtered by group.
func (s *ConfigService) GetAllConfigs(ctx context.Context, group string) ([]entity.DormConfig, error) {
	if group != "" {
		return s.configRepo.FindByGroup(ctx, group)
	}
	return s.configRepo.FindAll(ctx, "config_key ASC")
}

// GetConfigByKey returns a single config by its key.
func (s *ConfigService) GetConfigByKey(ctx context.Context, key string) (*entity.DormConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find config: %w", err)
	}
	return cfg, nil
}

// UpdateConfig updates a config's value by its key.
func (s *ConfigService) UpdateConfig(ctx context.Context, key, value string) error {
	// Verify config exists
	_, err := s.configRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("find config: %w", err)
	}

	if err := s.configRepo.UpdateByKey(ctx, key, value); err != nil {
		return fmt.Errorf("update config: %w", err)
	}

	s.log().Info("config updated", zap.String("key", key))
	return nil
}

// BatchUpdate applies multiple config updates atomically.
func (s *ConfigService) BatchUpdate(ctx context.Context, updates []dto.ConfigUpdateDTO) error {
	if len(updates) == 0 {
		return nil
	}

	m := make(map[string]string, len(updates))
	for _, u := range updates {
		m[u.ConfigKey] = u.ConfigValue
	}

	if err := s.configRepo.BatchUpdateByKey(ctx, m); err != nil {
		return fmt.Errorf("batch update: %w", err)
	}

	s.log().Info("batch updated", zap.Int("count", len(updates)))
	return nil
}

// ResetConfig resets a config to its default value.
func (s *ConfigService) ResetConfig(ctx context.Context, key string) (*entity.DormConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find config: %w", err)
	}

	if cfg.DefaultValue.Valid {
		if err := s.configRepo.UpdateByKey(ctx, key, cfg.DefaultValue.String); err != nil {
			return nil, fmt.Errorf("reset config: %w", err)
		}
		cfg.ConfigValue = cfg.DefaultValue.String
	}

	cfg.UpdatedAt = time.Now()
	s.log().Info("config reset to default", zap.String("key", key))
	return cfg, nil
}

// GetGroups returns all distinct config group names.
func (s *ConfigService) GetGroups(ctx context.Context) ([]string, error) {
	var groups []string
	err := s.configRepo.DB.SelectContext(ctx, &groups,
		"SELECT DISTINCT group_name FROM dorm_config WHERE group_name IS NOT NULL AND group_name != '' ORDER BY group_name")
	if err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}
	return groups, nil
}
