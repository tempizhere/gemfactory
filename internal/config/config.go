// Package config содержит загрузку и валидацию конфигурации.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config представляет конфигурацию приложения
type Config struct {
	// Database
	DatabaseURL string

	// Telegram
	BotToken      string
	AdminUsername string

	// Spotify
	SpotifyClientID     string
	SpotifyClientSecret string
	PlaylistURL         string

	// Health
	HealthPort         string
	HealthCheckEnabled bool

	// Logging
	LogLevel string

	// HTTP Client
	HTTPClientConfig HTTPClientConfig

	// Retry
	RetryConfig RetryConfig

	// Request delay
	RequestDelay time.Duration

	// Timezone
	Timezone string

	// App Data Directory
	AppDataDir string

	// Scraper
	ScraperConfig ScraperConfig

	// LLM
	LLMConfig LLMConfig
}

// HTTPClientConfig представляет конфигурацию HTTP клиента
type HTTPClientConfig struct {
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	DisableKeepAlives     bool
}

// RetryConfig представляет конфигурацию retry механизма
type RetryConfig struct {
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	// Загружаем .env файл если он существует
	if err := godotenv.Load(); err != nil {
		// Игнорируем ошибку если файл не найден
	}

	config := &Config{
		DatabaseURL:         getEnv("DB_DSN", ""),
		BotToken:            getEnv("BOT_TOKEN", ""),
		AdminUsername:       getEnv("ADMIN_USERNAME", ""),
		SpotifyClientID:     getEnv("SPOTIFY_CLIENT_ID", ""),
		SpotifyClientSecret: getEnv("SPOTIFY_CLIENT_SECRET", ""),
		PlaylistURL:         getEnv("PLAYLIST_URL", ""),
		HealthPort:          getEnv("HEALTH_PORT", "8080"),
		HealthCheckEnabled:  getEnvBool("HEALTH_CHECK_ENABLED", true),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		HTTPClientConfig: HTTPClientConfig{
			MaxIdleConns:          getEnvInt("HTTP_MAX_IDLE_CONNS", 100),
			MaxIdleConnsPerHost:   getEnvInt("HTTP_MAX_IDLE_CONNS_PER_HOST", 10),
			IdleConnTimeout:       getEnvDuration("HTTP_IDLE_CONN_TIMEOUT", 90*time.Second),
			TLSHandshakeTimeout:   getEnvDuration("HTTP_TLS_HANDSHAKE_TIMEOUT", 10*time.Second),
			ResponseHeaderTimeout: getEnvDuration("HTTP_RESPONSE_HEADER_TIMEOUT", 30*time.Second),
			DisableKeepAlives:     getEnvBool("HTTP_DISABLE_KEEP_ALIVES", false),
		},
		RetryConfig: RetryConfig{
			MaxRetries:        getEnvInt("RETRY_MAX_RETRIES", 3),
			InitialDelay:      getEnvDuration("RETRY_INITIAL_DELAY", 1*time.Second),
			MaxDelay:          getEnvDuration("RETRY_MAX_DELAY", 30*time.Second),
			BackoffMultiplier: getEnvFloat("RETRY_BACKOFF_MULTIPLIER", 2.0),
		},
		RequestDelay: getEnvDuration("REQUEST_DELAY", 2*time.Second),
		Timezone:     getEnv("TIMEZONE", "Europe/Moscow"),
		AppDataDir:   getEnv("APP_DATA_DIR", "./data"),
		ScraperConfig: ScraperConfig{
			HTTPClientConfig: ScraperHTTPClientConfig{
				MaxIdleConns:          getEnvInt("SCRAPER_MAX_IDLE_CONNS", 100),
				MaxIdleConnsPerHost:   getEnvInt("SCRAPER_MAX_IDLE_CONNS_PER_HOST", 10),
				IdleConnTimeout:       getEnvDuration("SCRAPER_IDLE_CONN_TIMEOUT", 90*time.Second),
				TLSHandshakeTimeout:   getEnvDuration("SCRAPER_TLS_HANDSHAKE_TIMEOUT", 10*time.Second),
				ResponseHeaderTimeout: getEnvDuration("SCRAPER_RESPONSE_HEADER_TIMEOUT", 30*time.Second),
				DisableKeepAlives:     getEnvBool("SCRAPER_DISABLE_KEEP_ALIVES", false),
			},
			RetryConfig: ScraperRetryConfig{
				MaxRetries:        getEnvInt("SCRAPER_MAX_RETRIES", 3),
				InitialDelay:      getEnvDuration("SCRAPER_INITIAL_DELAY", 1*time.Second),
				MaxDelay:          getEnvDuration("SCRAPER_MAX_DELAY", 30*time.Second),
				BackoffMultiplier: getEnvFloat("SCRAPER_BACKOFF_MULTIPLIER", 2.0),
			},
			RequestDelay: getEnvDuration("SCRAPER_REQUEST_DELAY", 2*time.Second),
		},
		LLMConfig: LLMConfig{
			BaseURL: getEnv("LLM_BASE_URL", "https://integrate.api.nvidia.com/v1"),
			APIKey:  getEnv("LLM_API_KEY", ""),
			Timeout: getEnvDuration("LLM_TIMEOUT", 2*time.Minute),
		},
	}

	// Валидация обязательных полей
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate проверяет конфигурацию
// GetAppDataDir возвращает директорию данных приложения
func (c *Config) GetAppDataDir() string {
	return c.AppDataDir
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DB_DSN is required")
	}

	if c.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}

	if c.AdminUsername == "" {
		return fmt.Errorf("ADMIN_USERNAME is required")
	}

	if c.SpotifyClientID == "" {
		return fmt.Errorf("SPOTIFY_CLIENT_ID is required")
	}

	if c.SpotifyClientSecret == "" {
		return fmt.Errorf("SPOTIFY_CLIENT_SECRET is required")
	}

	if c.PlaylistURL == "" {
		return fmt.Errorf("PLAYLIST_URL is required")
	}

	return nil
}

// getEnv получает переменную окружения с значением по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt получает переменную окружения как int
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration получает переменную окружения как time.Duration
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvBool получает переменную окружения как bool
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvFloat получает переменную окружения как float64
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// ScraperConfig представляет конфигурацию скрейпера
type ScraperConfig struct {
	HTTPClientConfig ScraperHTTPClientConfig
	RetryConfig      ScraperRetryConfig
	RequestDelay     time.Duration
}

// ScraperHTTPClientConfig представляет конфигурацию HTTP клиента для скрейпера
type ScraperHTTPClientConfig struct {
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	DisableKeepAlives     bool
}

// ScraperRetryConfig представляет конфигурацию повторов для скрейпера
type ScraperRetryConfig struct {
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// LLMConfig представляет конфигурацию LLM клиента
type LLMConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}
