package config

import (
    "fmt"
    "os"
    "strconv"
    "time"

    "github.com/joho/godotenv"
)

// Config holds the bot's configuration
type Config struct {
    BotToken      string
    AdminUsername string
    RequestDelay  time.Duration
    MaxRetries    int
    CacheDuration time.Duration
    WhitelistDir  string
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
    _ = godotenv.Load() // Загружаем .env

    cfg := &Config{
        BotToken:      os.Getenv("BOT_TOKEN"),
        AdminUsername: os.Getenv("ADMIN_USERNAME"),
        WhitelistDir:  os.Getenv("WHITELIST_DIR"),
    }

    if cfg.BotToken == "" {
        return nil, fmt.Errorf("BOT_TOKEN is required")
    }

    if cfg.AdminUsername == "" {
        cfg.AdminUsername = "fullofsarang" // Значение по умолчанию
    }

    requestDelayStr := os.Getenv("REQUEST_DELAY")
    if requestDelayStr == "" {
        cfg.RequestDelay = 3 * time.Second // Значение по умолчанию
    } else {
        var err error
        cfg.RequestDelay, err = time.ParseDuration(requestDelayStr)
        if err != nil {
            return nil, fmt.Errorf("invalid REQUEST_DELAY: %v", err)
        }
    }

    maxRetriesStr := os.Getenv("MAX_RETRIES")
    if maxRetriesStr == "" {
        cfg.MaxRetries = 3 // Значение по умолчанию
    } else {
        var err error
        cfg.MaxRetries, err = strconv.Atoi(maxRetriesStr)
        if err != nil || cfg.MaxRetries <= 0 {
            return nil, fmt.Errorf("invalid MAX_RETRIES: %v", err)
        }
    }

    cacheDurationStr := os.Getenv("CACHE_DURATION")
    if cacheDurationStr == "" {
        cfg.CacheDuration = 8 * time.Hour // Значение по умолчанию
    } else {
        var err error
        cfg.CacheDuration, err = time.ParseDuration(cacheDurationStr)
        if err != nil {
            return nil, fmt.Errorf("invalid CACHE_DURATION: %v", err)
        }
    }

    if cfg.WhitelistDir == "" {
        cfg.WhitelistDir = "internal/features/releasesbot/data" // Значение по умолчанию
    }
    // Проверяем, существует ли директория
    if _, err := os.Stat(cfg.WhitelistDir); os.IsNotExist(err) {
        return nil, fmt.Errorf("WHITELIST_DIR does not exist: %s", cfg.WhitelistDir)
    }

    return cfg, nil
}