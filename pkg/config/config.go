package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Config holds the bot's configuration
type Config struct {
	BotToken              string
	AdminUsername         string
	RequestDelay          time.Duration
	MaxRetries            int
	MaxConcurrentRequests int
	CacheDuration         time.Duration
	WhitelistDir          string
	Timezone              string
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load() // Загружаем .env

	cfg := &Config{
		BotToken:      os.Getenv("BOT_TOKEN"),
		AdminUsername: os.Getenv("ADMIN_USERNAME"),
		WhitelistDir:  os.Getenv("WHITELIST_DIR"),
		Timezone:      os.Getenv("TZ"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	if cfg.AdminUsername == "" {
		cfg.AdminUsername = "fullofsarang"
	}

	if cfg.Timezone == "" {
		cfg.Timezone = "Asia/Seoul"
	}

	requestDelayStr := os.Getenv("REQUEST_DELAY")
	if requestDelayStr == "" {
		cfg.RequestDelay = 3 * time.Second
	} else {
		var err error
		cfg.RequestDelay, err = time.ParseDuration(requestDelayStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REQUEST_DELAY: %w", err)
		}
	}

	maxRetriesStr := os.Getenv("MAX_RETRIES")
	if maxRetriesStr == "" {
		cfg.MaxRetries = 3
	} else {
		var err error
		cfg.MaxRetries, err = strconv.Atoi(maxRetriesStr)
		if err != nil || cfg.MaxRetries <= 0 {
			return nil, fmt.Errorf("invalid MAX_RETRIES: %w", err)
		}
	}

	maxConcurrentStr := os.Getenv("MAX_CONCURRENT_REQUESTS")
	if maxConcurrentStr == "" {
		cfg.MaxConcurrentRequests = 5
	} else {
		var err error
		cfg.MaxConcurrentRequests, err = strconv.Atoi(maxConcurrentStr)
		if err != nil || cfg.MaxConcurrentRequests <= 0 {
			return nil, fmt.Errorf("invalid MAX_CONCURRENT_REQUESTS: %w", err)
		}
	}

	cacheDurationStr := os.Getenv("CACHE_DURATION")
	if cacheDurationStr == "" {
		cfg.CacheDuration = 24 * time.Hour
	} else {
		var err error
		cfg.CacheDuration, err = time.ParseDuration(cacheDurationStr)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_DURATION: %w", err)
		}
	}

	if cfg.WhitelistDir == "" {
		cfg.WhitelistDir = "internal/telegrambot/releases/data"
	}
	if _, err := os.Stat(cfg.WhitelistDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("WHITELIST_DIR does not exist: %s", cfg.WhitelistDir)
	}

	return cfg, nil
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
