package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Config представляет конфигурацию приложения
type Config struct {
	BotToken      string
	AdminUsername string
	WhitelistDir  string
	Timezone      string

	// Настройки запросов
	RequestDelay          time.Duration
	MaxRetries            int
	MaxConcurrentRequests int

	// Настройки кэша
	CacheDuration time.Duration

	// Retry настройки
	RetryConfig RetryConfig

	// HTTP клиент настройки
	HTTPClientConfig HTTPClientConfig

	// Health check настройки
	HealthCheckPort    int
	HealthCheckEnabled bool

	// Rate limiting настройки
	RateLimitEnabled  bool
	RateLimitRequests int
	RateLimitWindow   time.Duration

	// Command cache настройки
	CommandCacheEnabled bool
	CommandCacheTTL     time.Duration

	// Логирование
	LogLevel string

	// Дополнительные настройки
	MetricsEnabled          bool
	HealthCheckInterval     time.Duration
	GracefulShutdownTimeout time.Duration
}

// Убеждаемся, что Config реализует ConfigInterface
var _ ConfigInterface = (*Config)(nil)

// RetryConfig конфигурация для retry механизма
type RetryConfig struct {
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// HTTPClientConfig конфигурация HTTP клиента
type HTTPClientConfig struct {
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	DisableKeepAlives     bool
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	config := &Config{}

	if err := config.loadBasicSettings(); err != nil {
		return nil, err
	}

	if err := config.loadRequestSettings(); err != nil {
		return nil, err
	}

	if err := config.loadCacheSettings(); err != nil {
		return nil, err
	}

	if err := config.loadRetrySettings(); err != nil {
		return nil, err
	}

	if err := config.loadHTTPSettings(); err != nil {
		return nil, err
	}

	if err := config.loadHealthCheckSettings(); err != nil {
		return nil, err
	}

	if err := config.loadRateLimitSettings(); err != nil {
		return nil, err
	}

	if err := config.loadCommandCacheSettings(); err != nil {
		return nil, err
	}

	if err := config.loadAdvancedSettings(); err != nil {
		return nil, err
	}

	return config, nil
}

// loadBasicSettings загружает базовые настройки
func (c *Config) loadBasicSettings() error {
	_ = godotenv.Load() // Загружаем .env

	c.BotToken = os.Getenv("BOT_TOKEN")
	c.AdminUsername = os.Getenv("ADMIN_USERNAME")
	c.WhitelistDir = os.Getenv("WHITELIST_DIR")
	c.Timezone = os.Getenv("TZ")

	// Валидация обязательных полей
	if c.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}

	// Значения по умолчанию
	if c.AdminUsername == "" {
		c.AdminUsername = "fullofsarang"
	}

	if c.Timezone == "" {
		c.Timezone = "Asia/Seoul"
	}

	return nil
}

// loadRequestSettings загружает настройки запросов
func (c *Config) loadRequestSettings() error {
	requestDelayStr := os.Getenv("REQUEST_DELAY")
	if requestDelayStr == "" {
		c.RequestDelay = 3 * time.Second
	} else {
		var err error
		c.RequestDelay, err = time.ParseDuration(requestDelayStr)
		if err != nil {
			return fmt.Errorf("invalid REQUEST_DELAY: %w", err)
		}
	}

	maxRetriesStr := os.Getenv("MAX_RETRIES")
	if maxRetriesStr == "" {
		c.MaxRetries = 3
	} else {
		var err error
		c.MaxRetries, err = strconv.Atoi(maxRetriesStr)
		if err != nil || c.MaxRetries <= 0 {
			return fmt.Errorf("invalid MAX_RETRIES: %w", err)
		}
	}

	maxConcurrentStr := os.Getenv("MAX_CONCURRENT_REQUESTS")
	if maxConcurrentStr == "" {
		c.MaxConcurrentRequests = 5
	} else {
		var err error
		c.MaxConcurrentRequests, err = strconv.Atoi(maxConcurrentStr)
		if err != nil || c.MaxConcurrentRequests <= 0 {
			return fmt.Errorf("invalid MAX_CONCURRENT_REQUESTS: %w", err)
		}
	}

	return nil
}

// loadCacheSettings загружает настройки кэша
func (c *Config) loadCacheSettings() error {
	cacheDurationStr := os.Getenv("CACHE_DURATION")
	if cacheDurationStr == "" {
		c.CacheDuration = 24 * time.Hour
	} else {
		var err error
		c.CacheDuration, err = time.ParseDuration(cacheDurationStr)
		if err != nil {
			return fmt.Errorf("invalid CACHE_DURATION: %w", err)
		}
	}
	return nil
}

// loadRetrySettings загружает настройки retry
func (c *Config) loadRetrySettings() error {
	maxRetries := getEnvInt("RETRY_MAX_ATTEMPTS", 3)
	initialDelay := getEnvDuration("RETRY_INITIAL_DELAY", 1*time.Second)
	maxDelay := getEnvDuration("RETRY_MAX_DELAY", 30*time.Second)
	backoffMultiplier := getEnvFloat("RETRY_BACKOFF_MULTIPLIER", 2.0)

	c.RetryConfig = RetryConfig{
		MaxRetries:        maxRetries,
		InitialDelay:      initialDelay,
		MaxDelay:          maxDelay,
		BackoffMultiplier: backoffMultiplier,
	}
	return nil
}

// loadHTTPSettings загружает настройки HTTP клиента
func (c *Config) loadHTTPSettings() error {
	// HTTP клиент настройки
	c.HTTPClientConfig.MaxIdleConns = getEnvInt("HTTP_MAX_IDLE_CONNS", 100)
	c.HTTPClientConfig.MaxIdleConnsPerHost = getEnvInt("HTTP_MAX_IDLE_CONNS_PER_HOST", 10)
	c.HTTPClientConfig.IdleConnTimeout = getEnvDuration("HTTP_IDLE_CONN_TIMEOUT", 90*time.Second)
	c.HTTPClientConfig.TLSHandshakeTimeout = getEnvDuration("HTTP_TLS_HANDSHAKE_TIMEOUT", 10*time.Second)
	c.HTTPClientConfig.ResponseHeaderTimeout = getEnvDuration("HTTP_RESPONSE_HEADER_TIMEOUT", 30*time.Second)

	disableKeepAlives := os.Getenv("HTTP_DISABLE_KEEP_ALIVES")
	c.HTTPClientConfig.DisableKeepAlives = disableKeepAlives == "true" || disableKeepAlives == "1"

	return nil
}

// loadHealthCheckSettings загружает настройки health check
func (c *Config) loadHealthCheckSettings() error {
	healthCheckPortStr := os.Getenv("HEALTH_CHECK_PORT")
	if healthCheckPortStr == "" {
		c.HealthCheckPort = 8080
	} else {
		var err error
		c.HealthCheckPort, err = strconv.Atoi(healthCheckPortStr)
		if err != nil || c.HealthCheckPort <= 0 {
			return fmt.Errorf("invalid HEALTH_CHECK_PORT: %w", err)
		}
	}

	healthCheckEnabledStr := os.Getenv("HEALTH_CHECK_ENABLED")
	if healthCheckEnabledStr == "" {
		c.HealthCheckEnabled = true
	} else {
		var err error
		c.HealthCheckEnabled, err = strconv.ParseBool(healthCheckEnabledStr)
		if err != nil {
			return fmt.Errorf("invalid HEALTH_CHECK_ENABLED: %w", err)
		}
	}

	return nil
}

// loadRateLimitSettings загружает настройки rate limiting
func (c *Config) loadRateLimitSettings() error {
	rateLimitEnabledStr := os.Getenv("RATE_LIMIT_ENABLED")
	if rateLimitEnabledStr == "" {
		c.RateLimitEnabled = true
	} else {
		var err error
		c.RateLimitEnabled, err = strconv.ParseBool(rateLimitEnabledStr)
		if err != nil {
			return fmt.Errorf("invalid RATE_LIMIT_ENABLED: %w", err)
		}
	}

	rateLimitRequestsStr := os.Getenv("RATE_LIMIT_REQUESTS")
	if rateLimitRequestsStr == "" {
		c.RateLimitRequests = 10
	} else {
		var err error
		c.RateLimitRequests, err = strconv.Atoi(rateLimitRequestsStr)
		if err != nil || c.RateLimitRequests <= 0 {
			return fmt.Errorf("invalid RATE_LIMIT_REQUESTS: %w", err)
		}
	}

	rateLimitWindowStr := os.Getenv("RATE_LIMIT_WINDOW")
	if rateLimitWindowStr == "" {
		c.RateLimitWindow = 60 * time.Second
	} else {
		var err error
		c.RateLimitWindow, err = time.ParseDuration(rateLimitWindowStr)
		if err != nil {
			return fmt.Errorf("invalid RATE_LIMIT_WINDOW: %w", err)
		}
	}

	return nil
}

// loadCommandCacheSettings загружает настройки command cache
func (c *Config) loadCommandCacheSettings() error {
	commandCacheEnabledStr := os.Getenv("COMMAND_CACHE_ENABLED")
	if commandCacheEnabledStr == "" {
		c.CommandCacheEnabled = true
	} else {
		var err error
		c.CommandCacheEnabled, err = strconv.ParseBool(commandCacheEnabledStr)
		if err != nil {
			return fmt.Errorf("invalid COMMAND_CACHE_ENABLED: %w", err)
		}
	}

	commandCacheTTLStr := os.Getenv("COMMAND_CACHE_TTL")
	if commandCacheTTLStr == "" {
		c.CommandCacheTTL = 5 * time.Minute
	} else {
		var err error
		c.CommandCacheTTL, err = time.ParseDuration(commandCacheTTLStr)
		if err != nil {
			return fmt.Errorf("invalid COMMAND_CACHE_TTL: %w", err)
		}
	}

	return nil
}

// loadAdvancedSettings загружает продвинутые настройки
func (c *Config) loadAdvancedSettings() error {
	// Метрики
	metricsEnabled := os.Getenv("METRICS_ENABLED")
	c.MetricsEnabled = metricsEnabled == "true" || metricsEnabled == "1"

	// Health check
	healthCheckInterval := os.Getenv("HEALTH_CHECK_INTERVAL")
	if healthCheckInterval == "" {
		c.HealthCheckInterval = 30 * time.Second
	} else {
		var err error
		c.HealthCheckInterval, err = time.ParseDuration(healthCheckInterval)
		if err != nil {
			return fmt.Errorf("invalid HEALTH_CHECK_INTERVAL: %w", err)
		}
	}

	// Graceful shutdown timeout
	gracefulShutdownTimeout := os.Getenv("GRACEFUL_SHUTDOWN_TIMEOUT")
	if gracefulShutdownTimeout == "" {
		c.GracefulShutdownTimeout = 30 * time.Second
	} else {
		var err error
		c.GracefulShutdownTimeout, err = time.ParseDuration(gracefulShutdownTimeout)
		if err != nil {
			return fmt.Errorf("invalid GRACEFUL_SHUTDOWN_TIMEOUT: %w", err)
		}
	}

	return nil
}

// validateWhitelistDir валидирует директорию whitelist
func (c *Config) validateWhitelistDir() error {
	if c.WhitelistDir == "" {
		c.WhitelistDir = "internal/telegrambot/releases/data"
	}
	if _, err := os.Stat(c.WhitelistDir); os.IsNotExist(err) {
		return fmt.Errorf("WHITELIST_DIR does not exist: %s", c.WhitelistDir)
	}
	return nil
}

// LoadLocation loads the timezone specified in the config
func (c *Config) LoadLocation(logger *zap.Logger) *time.Location {
	if logger == nil {
		logger = zap.NewNop()
	}
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		logger.Error("Failed to load timezone, falling back to UTC",
			zap.String("timezone", c.Timezone),
			zap.Error(err))
		return time.UTC
	}
	logger.Info("Timezone loaded", zap.String("timezone", c.Timezone))
	return loc
}

// GetBotToken возвращает токен бота
func (c *Config) GetBotToken() string {
	return c.BotToken
}

// GetAdminUsername возвращает имя администратора
func (c *Config) GetAdminUsername() string {
	return c.AdminUsername
}

// GetWhitelistDir возвращает директорию белых списков
func (c *Config) GetWhitelistDir() string {
	return c.WhitelistDir
}

// GetTimezone возвращает временную зону
func (c *Config) GetTimezone() string {
	return c.Timezone
}

// GetRateLimitWindow возвращает окно rate limiting
func (c *Config) GetRateLimitWindow() time.Duration {
	return c.RateLimitWindow
}

// GetMaxConcurrentRequests возвращает максимальное количество одновременных запросов
func (c *Config) GetMaxConcurrentRequests() int {
	return c.MaxConcurrentRequests
}

// GetCommandCacheEnabled возвращает статус включения кэша команд
func (c *Config) GetCommandCacheEnabled() bool {
	return c.CommandCacheEnabled
}

// GetCommandCacheTTL возвращает TTL кэша команд
func (c *Config) GetCommandCacheTTL() time.Duration {
	return c.CommandCacheTTL
}

// GetRateLimitEnabled возвращает статус включения rate limiting
func (c *Config) GetRateLimitEnabled() bool {
	return c.RateLimitEnabled
}

// GetRateLimitRequests возвращает лимит запросов
func (c *Config) GetRateLimitRequests() int {
	return c.RateLimitRequests
}

// GetHealthCheckEnabled возвращает статус включения health check
func (c *Config) GetHealthCheckEnabled() bool {
	return c.HealthCheckEnabled
}

// GetHealthCheckPort возвращает порт health check
func (c *Config) GetHealthCheckPort() int {
	return c.HealthCheckPort
}

// Вспомогательные функции для парсинга переменных окружения
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
