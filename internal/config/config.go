// Package config реализует загрузку и хранение конфигурации приложения.
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
	AppDataDir    string
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

	// Playlist настройки
	PlaylistCSVPath string

	// Информация о приложении
	AppName    string
	AppVersion string
	BuildTime  string
	GitCommit  string
}

// Interface определяет полный интерфейс конфигурации
type Interface interface {
	// Основные настройки бота
	GetBotToken() string
	GetAdminUsername() string
	GetAppDataDir() string
	GetTimezone() string
	LoadLocation(logger *zap.Logger) *time.Location

	// Cache настройки
	GetCacheDuration() time.Duration
	GetCommandCacheEnabled() bool
	GetCommandCacheTTL() time.Duration

	// Rate limiting настройки
	GetRateLimitEnabled() bool
	GetRateLimitRequests() int
	GetRateLimitWindow() time.Duration

	// Health check настройки
	GetHealthCheckEnabled() bool
	GetHealthCheckPort() int

	// Metrics настройки
	GetMetricsEnabled() bool

	// Performance настройки
	GetMaxConcurrentRequests() int
	GetRequestDelay() time.Duration
	GetMaxRetries() int

	// Graceful shutdown настройки
	GetGracefulShutdownTimeout() time.Duration

	// Playlist настройки
	GetPlaylistCSVPath() string

	// Информация о приложении
	GetAppName() string
	GetAppVersion() string
	GetBuildTime() string
	GetGitCommit() string

	// Валидация
	Validate() error
	ValidateEnvironment() error
}

var _ Interface = (*Config)(nil)

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

	if err := config.loadAppInfo(); err != nil {
		return nil, err
	}

	// Валидация конфигурации
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	// Обязательные поля
	if c.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}

	// Валидация числовых значений
	if c.MaxRetries <= 0 {
		return fmt.Errorf("MAX_RETRIES must be positive, got: %d", c.MaxRetries)
	}

	if c.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("MAX_CONCURRENT_REQUESTS must be positive, got: %d", c.MaxConcurrentRequests)
	}

	if c.CacheDuration <= 0 {
		return fmt.Errorf("CACHE_DURATION must be positive, got: %v", c.CacheDuration)
	}

	// Валидация retry настроек
	if c.RetryConfig.MaxRetries < 0 {
		return fmt.Errorf("RETRY_MAX_ATTEMPTS cannot be negative, got: %d", c.RetryConfig.MaxRetries)
	}

	if c.RetryConfig.InitialDelay <= 0 {
		return fmt.Errorf("RETRY_INITIAL_DELAY must be positive, got: %v", c.RetryConfig.InitialDelay)
	}

	if c.RetryConfig.MaxDelay <= 0 {
		return fmt.Errorf("RETRY_MAX_DELAY must be positive, got: %v", c.RetryConfig.MaxDelay)
	}

	if c.RetryConfig.BackoffMultiplier <= 0 {
		return fmt.Errorf("RETRY_BACKOFF_MULTIPLIER must be positive, got: %f", c.RetryConfig.BackoffMultiplier)
	}

	// Валидация health check настроек
	if c.HealthCheckEnabled && (c.HealthCheckPort <= 0 || c.HealthCheckPort > 65535) {
		return fmt.Errorf("HEALTH_CHECK_PORT must be between 1 and 65535, got: %d", c.HealthCheckPort)
	}

	// Валидация rate limiting настроек
	if c.RateLimitEnabled {
		if c.RateLimitRequests <= 0 {
			return fmt.Errorf("RATE_LIMIT_REQUESTS must be positive, got: %d", c.RateLimitRequests)
		}

		if c.RateLimitWindow <= 0 {
			return fmt.Errorf("RATE_LIMIT_WINDOW must be positive, got: %v", c.RateLimitWindow)
		}
	}

	// Валидация command cache настроек
	if c.CommandCacheEnabled && c.CommandCacheTTL <= 0 {
		return fmt.Errorf("COMMAND_CACHE_TTL must be positive when cache is enabled, got: %v", c.CommandCacheTTL)
	}

	return nil
}

// loadBasicSettings загружает базовые настройки
func (c *Config) loadBasicSettings() error {
	_ = godotenv.Load() // Загружаем .env

	c.BotToken = os.Getenv("BOT_TOKEN")
	c.AdminUsername = os.Getenv("ADMIN_USERNAME")
	c.AppDataDir = os.Getenv("APP_DATA_DIR")
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
		c.GracefulShutdownTimeout = 10 * time.Second
	} else {
		var err error
		c.GracefulShutdownTimeout, err = time.ParseDuration(gracefulShutdownTimeout)
		if err != nil {
			return fmt.Errorf("invalid GRACEFUL_SHUTDOWN_TIMEOUT: %w", err)
		}
	}

	// Playlist CSV path (необязательный)
	playlistCSVPath := os.Getenv("PLAYLIST_CSV_PATH")
	if playlistCSVPath == "" {
		c.PlaylistCSVPath = "" // Пустой путь - плейлист не загружается автоматически
	} else {
		c.PlaylistCSVPath = playlistCSVPath
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

// GetAppDataDir возвращает директорию данных приложения
func (c *Config) GetAppDataDir() string {
	return c.AppDataDir
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

// GetCacheDuration возвращает длительность кэша
func (c *Config) GetCacheDuration() time.Duration {
	return c.CacheDuration
}

// GetRequestDelay возвращает задержку между запросами
func (c *Config) GetRequestDelay() time.Duration {
	return c.RequestDelay
}

// GetMaxRetries возвращает максимальное количество повторов
func (c *Config) GetMaxRetries() int {
	return c.MaxRetries
}

// GetMetricsEnabled возвращает статус включения метрик
func (c *Config) GetMetricsEnabled() bool {
	return c.MetricsEnabled
}

// GetGracefulShutdownTimeout возвращает таймаут для graceful shutdown
func (c *Config) GetGracefulShutdownTimeout() time.Duration {
	return c.GracefulShutdownTimeout
}

// GetPlaylistCSVPath возвращает путь к CSV файлу плейлиста
func (c *Config) GetPlaylistCSVPath() string {
	return c.PlaylistCSVPath
}

// loadAppInfo загружает информацию о приложении
func (c *Config) loadAppInfo() error {
	c.AppName = "GemFactory Telegram Bot"

	if v := os.Getenv("APP_VERSION"); v != "" {
		c.AppVersion = v
	} else {
		c.AppVersion = "1.0.0"
	}

	if t := os.Getenv("BUILD_TIME"); t != "" {
		c.BuildTime = t
	} else {
		c.BuildTime = time.Now().Format(time.RFC3339)
	}

	if g := os.Getenv("GIT_COMMIT"); g != "" {
		c.GitCommit = g
	} else {
		c.GitCommit = "unknown"
	}

	return nil
}

// GetAppName возвращает имя приложения
func (c *Config) GetAppName() string {
	return c.AppName
}

// GetAppVersion возвращает версию приложения
func (c *Config) GetAppVersion() string {
	return c.AppVersion
}

// GetBuildTime возвращает время сборки
func (c *Config) GetBuildTime() string {
	return c.BuildTime
}

// GetGitCommit возвращает Git commit hash
func (c *Config) GetGitCommit() string {
	return c.GitCommit
}

// ValidateEnvironment проверяет критические переменные окружения
func (c *Config) ValidateEnvironment() error {
	requiredEnvs := []string{
		"BOT_TOKEN",
	}

	var missingEnvs []string
	for _, env := range requiredEnvs {
		if os.Getenv(env) == "" {
			missingEnvs = append(missingEnvs, env)
		}
	}

	if len(missingEnvs) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missingEnvs)
	}

	return nil
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
