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

// LoadConfigFromDB загружает конфигурацию из базы данных
func (cl *ConfigLoader) LoadConfigFromDB(cfg *Config) {
	// ADMIN_USERNAME
	cl.LoadConfigValueWithSetter(cfg.AdminUsername, "ADMIN_USERNAME", func(value string) {
		cfg.AdminUsername = value
	})

	// LLM_API_KEY
	cl.LoadConfigValueWithSetter(cfg.LLMConfig.APIKey, "LLM_API_KEY", func(value string) {
		cfg.LLMConfig.APIKey = value
	})

	// BOT_TOKEN
	cl.LoadConfigValueWithSetter(cfg.BotToken, "BOT_TOKEN", func(value string) {
		cfg.BotToken = value
	})

	// SPOTIFY_CLIENT_ID
	cl.LoadConfigValueWithSetter(cfg.SpotifyClientID, "SPOTIFY_CLIENT_ID", func(value string) {
		cfg.SpotifyClientID = value
	})

	// SPOTIFY_CLIENT_SECRET
	cl.LoadConfigValueWithSetter(cfg.SpotifyClientSecret, "SPOTIFY_CLIENT_SECRET", func(value string) {
		cfg.SpotifyClientSecret = value
	})

	// PLAYLIST_URL
	cl.LoadConfigValueWithSetter(cfg.PlaylistURL, "PLAYLIST_URL", func(value string) {
		cfg.PlaylistURL = value
	})
}
