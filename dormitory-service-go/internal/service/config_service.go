package service

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
)

// ConfigService handles configuration CRUD operations.
type ConfigService struct {
	configRepo *repository.ConfigRepository
}

// NewConfigService creates a new ConfigService.
func NewConfigService(configRepo *repository.ConfigRepository) *ConfigService {
	return &ConfigService{
		configRepo: configRepo,
	}
}

// GetAllConfigs returns all configs, optionally filtered by group.
func (s *ConfigService) GetAllConfigs(group string) ([]entity.DormConfig, error) {
	if group != "" {
		return s.configRepo.FindByGroup(group)
	}
	return s.configRepo.FindAll("config_key ASC")
}

// GetConfigByKey returns a single config by its key.
func (s *ConfigService) GetConfigByKey(key string) (*entity.DormConfig, error) {
	cfg, err := s.configRepo.FindByKey(key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find config: %w", err)
	}
	return cfg, nil
}

// UpdateConfig updates a config's value by its key.
func (s *ConfigService) UpdateConfig(key, value string) error {
	// Verify config exists
	_, err := s.configRepo.FindByKey(key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("find config: %w", err)
	}

	if err := s.configRepo.UpdateByKey(key, value); err != nil {
		return fmt.Errorf("update config: %w", err)
	}

	log.Printf("[ConfigService] Config updated: key=%s", key)
	return nil
}

// BatchUpdate applies multiple config updates atomically.
func (s *ConfigService) BatchUpdate(updates []dto.ConfigUpdateDTO) error {
	if len(updates) == 0 {
		return nil
	}
	for _, u := range updates {
		if err := s.UpdateConfig(u.ConfigKey, u.ConfigValue); err != nil {
			return fmt.Errorf("batch update %s: %w", u.ConfigKey, err)
		}
	}
	log.Printf("[ConfigService] Batch updated %d configs", len(updates))
	return nil
}

// ResetConfig resets a config to its default value.
func (s *ConfigService) ResetConfig(key string) (*entity.DormConfig, error) {
	cfg, err := s.configRepo.FindByKey(key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find config: %w", err)
	}

	if cfg.DefaultValue.Valid {
		if err := s.configRepo.UpdateByKey(key, cfg.DefaultValue.String); err != nil {
			return nil, fmt.Errorf("reset config: %w", err)
		}
		cfg.ConfigValue = cfg.DefaultValue.String
	}

	cfg.UpdatedAt = time.Now()
	log.Printf("[ConfigService] Config reset to default: key=%s", key)
	return cfg, nil
}

// GetGroups returns all distinct config group names.
func (s *ConfigService) GetGroups() ([]string, error) {
	var groups []string
	err := s.configRepo.DB.Select(&groups,
		"SELECT DISTINCT group_name FROM dorm_config WHERE group_name IS NOT NULL AND group_name != '' ORDER BY group_name")
	if err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}
	return groups, nil
}
