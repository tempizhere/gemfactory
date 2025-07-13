package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				BotToken:              "test-token",
				MaxRetries:            3,
				MaxConcurrentRequests: 5,
				CacheDuration:         time.Hour,
				RetryConfig: RetryConfig{
					MaxRetries:        3,
					InitialDelay:      time.Second,
					MaxDelay:          30 * time.Second,
					BackoffMultiplier: 2.0,
				},
				HealthCheckEnabled:  true,
				HealthCheckPort:     8080,
				RateLimitEnabled:    true,
				RateLimitRequests:   10,
				RateLimitWindow:     time.Minute,
				CommandCacheEnabled: true,
				CommandCacheTTL:     5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing bot token",
			config: &Config{
				MaxRetries:            3,
				MaxConcurrentRequests: 5,
				CacheDuration:         time.Hour,
				RetryConfig: RetryConfig{
					MaxRetries:        3,
					InitialDelay:      time.Second,
					MaxDelay:          30 * time.Second,
					BackoffMultiplier: 2.0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid health check port",
			config: &Config{
				BotToken:              "test-token",
				MaxRetries:            3,
				MaxConcurrentRequests: 5,
				CacheDuration:         time.Hour,
				RetryConfig: RetryConfig{
					MaxRetries:        3,
					InitialDelay:      time.Second,
					MaxDelay:          30 * time.Second,
					BackoffMultiplier: 2.0,
				},
				HealthCheckEnabled: true,
				HealthCheckPort:    70000, // Invalid port
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// safeSetEnv безопасно устанавливает переменную окружения
func safeSetEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set env var %s: %v", key, err)
	}
}

// safeUnsetEnv безопасно удаляет переменную окружения
func safeUnsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Failed to unset env var %s: %v", key, err)
	}
}

func TestLoad(t *testing.T) {
	// Сохраняем текущие env vars
	originalToken := os.Getenv("BOT_TOKEN")
	defer func() {
		if originalToken != "" {
			safeSetEnv(t, "BOT_TOKEN", originalToken)
		} else {
			safeUnsetEnv(t, "BOT_TOKEN")
		}
	}()

	t.Run("missing required env var", func(t *testing.T) {
		safeUnsetEnv(t, "BOT_TOKEN")
		_, err := Load()
		if err == nil {
			t.Error("Load() should fail when BOT_TOKEN is missing")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		safeSetEnv(t, "BOT_TOKEN", "test-token")
		config, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}
		if config.BotToken != "test-token" {
			t.Errorf("BotToken = %v, want test-token", config.BotToken)
		}
	})
}

func TestConfig_Interface(t *testing.T) {
	config := &Config{
		BotToken:      "test_token",
		AdminUsername: "test_admin",
		WhitelistDir:  "/test/dir",
		Timezone:      "Europe/Moscow",
	}

	// Test Interface methods
	assert.Equal(t, "test_token", config.GetBotToken())
	assert.Equal(t, "test_admin", config.GetAdminUsername())
	assert.Equal(t, "/test/dir", config.GetWhitelistDir())
	assert.Equal(t, "Europe/Moscow", config.GetTimezone())
	assert.Equal(t, time.Duration(0), config.GetCacheDuration())
	assert.False(t, config.GetCommandCacheEnabled())
	assert.Equal(t, time.Duration(0), config.GetCommandCacheTTL())
	assert.False(t, config.GetRateLimitEnabled())
	assert.Equal(t, 0, config.GetRateLimitRequests())
	assert.Equal(t, time.Duration(0), config.GetRateLimitWindow())
	assert.False(t, config.GetHealthCheckEnabled())
	assert.Equal(t, 0, config.GetHealthCheckPort())
	assert.False(t, config.GetMetricsEnabled())
	assert.Equal(t, 0, config.GetMaxConcurrentRequests())
	assert.Equal(t, time.Duration(0), config.GetRequestDelay())
	assert.Equal(t, 0, config.GetMaxRetries())
}
