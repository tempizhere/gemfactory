// Package repository содержит репозитории для работы с базой данных.
package repository

import (
	"context"
	"fmt"
	"gemfactory/internal/model"
	"strings"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ConfigRepository реализует интерфейс для работы с конфигурацией
type ConfigRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewConfigRepository создает новый репозиторий конфигурации
func NewConfigRepository(db *bun.DB, logger *zap.Logger) *ConfigRepository {
	return &ConfigRepository{
		db:     db,
		logger: logger,
	}
}

// Get возвращает конфигурацию по ключу
func (r *ConfigRepository) Get(key string) (*model.Config, error) {
	ctx := context.Background()
	config := new(model.Config)

	// Устанавливаем search_path для этого запроса
	_, err := r.db.ExecContext(ctx, "SET search_path TO gemfactory, public")
	if err != nil {
		r.logger.Warn("Failed to set search_path", zap.Error(err))
	}

	err = r.db.NewSelect().
		Model(config).
		Where("key = ?", key).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan config: %w", err)
	}

	return config, nil
}

// GetAll возвращает всю конфигурацию
func (r *ConfigRepository) GetAll() ([]model.Config, error) {
	ctx := context.Background()
	var configs []model.Config

	// Устанавливаем search_path для этого запроса
	_, err := r.db.ExecContext(ctx, "SET search_path TO gemfactory, public")
	if err != nil {
		r.logger.Warn("Failed to set search_path", zap.Error(err))
	}

	err = r.db.NewSelect().
		Model(&configs).
		Order("key ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query config: %w", err)
	}

	return configs, nil
}

// Set устанавливает значение конфигурации
func (r *ConfigRepository) Set(key, value string) error {
	ctx := context.Background()

	// Устанавливаем search_path для этого запроса
	_, err := r.db.ExecContext(ctx, "SET search_path TO gemfactory, public")
	if err != nil {
		r.logger.Warn("Failed to set search_path", zap.Error(err))
	}

	config := &model.Config{
		Key:   key,
		Value: value,
	}

	_, err = r.db.NewInsert().
		Model(config).
		On("CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	return nil
}

// Delete удаляет конфигурацию
func (r *ConfigRepository) Delete(key string) error {
	ctx := context.Background()

	// Устанавливаем search_path для этого запроса
	_, err := r.db.ExecContext(ctx, "SET search_path TO gemfactory, public")
	if err != nil {
		r.logger.Warn("Failed to set search_path", zap.Error(err))
	}

	_, err = r.db.NewDelete().
		Model((*model.Config)(nil)).
		Where("key = ?", key).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	return nil
}

// Reset сбрасывает конфигурацию к значениям по умолчанию
func (r *ConfigRepository) Reset() error {
	ctx := context.Background()

	// Устанавливаем search_path для этого запроса
	_, err := r.db.ExecContext(ctx, "SET search_path TO gemfactory, public")
	if err != nil {
		r.logger.Warn("Failed to set search_path", zap.Error(err))
	}

	_, err = r.db.NewDelete().
		Model((*model.Config)(nil)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	defaultConfig := r.GetDefaultConfig()
	for key, value := range defaultConfig {
		err := r.Set(key, value)
		if err != nil {
			return fmt.Errorf("failed to set default config %s: %w", key, err)
		}
	}

	return nil
}

// GetDefaultConfig возвращает конфигурацию по умолчанию
func (r *ConfigRepository) GetDefaultConfig() map[string]string {
	return map[string]string{
		"RATE_LIMIT_REQUESTS":   "10",
		"RATE_LIMIT_WINDOW":     "60",
		"SCRAPER_DELAY":         "1",
		"SCRAPER_TIMEOUT":       "30",
		"LOG_LEVEL":             "info",
		"BOT_TOKEN":             "",
		"ADMIN_USERNAME":        "",
		"SPOTIFY_CLIENT_ID":     "",
		"SPOTIFY_CLIENT_SECRET": "",
		"PLAYLIST_URL":          "",
		"DB_DSN":                "",
		"HEALTH_PORT":           "8080",
		"LLM_API_KEY":           "",
	}
}

// GetAllAsString возвращает всю конфигурацию в виде строки
func (r *ConfigRepository) GetAllAsString() (string, error) {
	configs, err := r.GetAll()
	if err != nil {
		return "", err
	}

	var result strings.Builder
	result.WriteString("📋 Текущая конфигурация:\n\n")

	for _, config := range configs {
		result.WriteString(fmt.Sprintf("🔧 %s: %s\n", config.Key, config.Value))
	}

	return result.String(), nil
}
