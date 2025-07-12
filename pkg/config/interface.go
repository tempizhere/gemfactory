package config

import (
	"time"

	"go.uber.org/zap"
)

// ConfigInterface определяет интерфейс для конфигурации
type ConfigInterface interface {
	// LoadLocation загружает временную зону
	LoadLocation(logger *zap.Logger) *time.Location

	// GetBotToken возвращает токен бота
	GetBotToken() string

	// GetAdminUsername возвращает имя администратора
	GetAdminUsername() string

	// GetWhitelistDir возвращает директорию белых списков
	GetWhitelistDir() string

	// GetTimezone возвращает временную зону
	GetTimezone() string

	// GetRateLimitWindow возвращает окно rate limiting
	GetRateLimitWindow() time.Duration

	// GetMaxConcurrentRequests возвращает максимальное количество одновременных запросов
	GetMaxConcurrentRequests() int

	// GetCommandCacheEnabled возвращает статус включения кэша команд
	GetCommandCacheEnabled() bool

	// GetCommandCacheTTL возвращает TTL кэша команд
	GetCommandCacheTTL() time.Duration

	// GetRateLimitEnabled возвращает статус включения rate limiting
	GetRateLimitEnabled() bool

	// GetRateLimitRequests возвращает лимит запросов
	GetRateLimitRequests() int

	// GetHealthCheckEnabled возвращает статус включения health check
	GetHealthCheckEnabled() bool

	// GetHealthCheckPort возвращает порт health check
	GetHealthCheckPort() int
}
