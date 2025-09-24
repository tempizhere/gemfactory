// Package config содержит утилиты для загрузки конфигурации
package config

import (
	"go.uber.org/zap"
)

// ConfigLoader представляет загрузчик конфигурации
type ConfigLoader struct {
	configService ConfigServiceInterface
	logger        *zap.Logger
}

// ConfigServiceInterface определяет интерфейс для работы с конфигурацией
type ConfigServiceInterface interface {
	Get(key string) (string, error)
}

// NewConfigLoader создает новый загрузчик конфигурации
func NewConfigLoader(configService ConfigServiceInterface, logger *zap.Logger) *ConfigLoader {
	return &ConfigLoader{
		configService: configService,
		logger:        logger,
	}
}

// LoadConfigValue загружает значение конфигурации с приоритетом: env > база данных
func (cl *ConfigLoader) LoadConfigValue(envValue, configKey string) string {
	if envValue == "" {
		if dbValue, err := cl.configService.Get(configKey); err == nil && dbValue != "" {
			cl.logger.Info("Loaded "+configKey+" from database", zap.String("value", dbValue))
			return dbValue
		} else {
			cl.logger.Debug("Failed to load "+configKey+" from database", zap.Error(err))
			return ""
		}
	} else {
		cl.logger.Info("Using "+configKey+" from environment variables", zap.String("value", envValue))
		return envValue
	}
}

// LoadConfigValueWithSetter загружает значение конфигурации и устанавливает его через setter
func (cl *ConfigLoader) LoadConfigValueWithSetter(envValue, configKey string, setter func(string)) string {
	value := cl.LoadConfigValue(envValue, configKey)
	if value != "" {
		setter(value)
	}
	return value
}
