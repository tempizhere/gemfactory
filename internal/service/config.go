// Package service содержит бизнес-логику приложения.
package service

import (
	"fmt"
	"gemfactory/internal/model"
	"gemfactory/internal/storage/repository"
	"strings"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ConfigService содержит бизнес-логику для работы с конфигурацией
type ConfigService struct {
	repo   model.ConfigRepository
	logger *zap.Logger
}

// NewConfigService создает новый сервис конфигурации
func NewConfigService(db *bun.DB, logger *zap.Logger) *ConfigService {
	return &ConfigService{
		repo:   repository.NewConfigRepository(db, logger),
		logger: logger,
	}
}

// Set устанавливает значение конфигурации
func (s *ConfigService) Set(key, value string) error {
	err := s.repo.Set(key, value)
	if err != nil {
		return fmt.Errorf("failed to set config %s: %w", key, err)
	}

	s.logger.Info("Config updated", zap.String("key", key), zap.String("value", value))
	return nil
}

// Get возвращает значение конфигурации
func (s *ConfigService) Get(key string) (string, error) {
	config, err := s.repo.Get(key)
	if err != nil {
		return "", fmt.Errorf("failed to get config %s: %w", key, err)
	}

	if config == nil {
		return "", fmt.Errorf("config %s not found", key)
	}

	return config.Value, nil
}

// GetAll возвращает всю конфигурацию
func (s *ConfigService) GetAll() (string, error) {
	configs, err := s.repo.GetAll()
	if err != nil {
		return "", fmt.Errorf("failed to get all configs: %w", err)
	}

	sensitiveKeys := map[string]bool{
		"BOT_TOKEN":             true,
		"LLM_API_KEY":           true,
		"SPOTIFY_CLIENT_ID":     true,
		"SPOTIFY_CLIENT_SECRET": true,
	}

	var result strings.Builder
	result.WriteString("📋 Текущая конфигурация:\n\n")

	for _, config := range configs {
		var value string
		if sensitiveKeys[config.Key] {
			value = "🔒 [СКРЫТО В ЦЕЛЯХ БЕЗОПАСНОСТИ - СМОТРИТЕ В ОКРУЖЕНИИ]"
		} else {
			value = config.Value
		}

		result.WriteString(fmt.Sprintf("🔧 <b>%s</b>: %s\n", config.Key, value))
		if config.Description != "" {
			result.WriteString(fmt.Sprintf("   📝 %s\n", config.Description))
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}

// Reset сбрасывает конфигурацию к значениям по умолчанию
func (s *ConfigService) Reset() error {
	err := s.repo.Reset()
	if err != nil {
		return fmt.Errorf("failed to reset config: %w", err)
	}

	s.logger.Info("Config reset to default values")
	return nil
}

// GetInt возвращает значение конфигурации как int
func (s *ConfigService) GetInt(key string) (int, error) {
	value, err := s.Get(key)
	if err != nil {
		return 0, err
	}

	var intValue int
	_, err = fmt.Sscanf(value, "%d", &intValue)
	if err != nil {
		return 0, fmt.Errorf("failed to parse config %s as int: %w", key, err)
	}

	return intValue, nil
}

// GetBool возвращает значение конфигурации как bool
func (s *ConfigService) GetBool(key string) (bool, error) {
	value, err := s.Get(key)
	if err != nil {
		return false, err
	}

	return strings.ToLower(value) == "true", nil
}

// GetFloat возвращает значение конфигурации как float64
func (s *ConfigService) GetFloat(key string) (float64, error) {
	value, err := s.Get(key)
	if err != nil {
		return 0, err
	}

	var floatValue float64
	_, err = fmt.Sscanf(value, "%f", &floatValue)
	if err != nil {
		return 0, fmt.Errorf("failed to parse config %s as float: %w", key, err)
	}

	return floatValue, nil
}

// SetInt устанавливает значение конфигурации как int
func (s *ConfigService) SetInt(key string, value int) error {
	return s.Set(key, fmt.Sprintf("%d", value))
}

// SetBool устанавливает значение конфигурации как bool
func (s *ConfigService) SetBool(key string, value bool) error {
	return s.Set(key, fmt.Sprintf("%t", value))
}

// SetFloat устанавливает значение конфигурации как float64
func (s *ConfigService) SetFloat(key string, value float64) error {
	return s.Set(key, fmt.Sprintf("%f", value))
}

// GetConfigValue возвращает значение конфигурации (алиас для Get)
func (s *ConfigService) GetConfigValue(key string) (string, error) {
	return s.Get(key)
}
