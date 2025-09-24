// Package service содержит бизнес-логику приложения.
package service

import (
	"context"
	"gemfactory/internal/model"
	"time"

	"go.uber.org/zap"
)

// ConfigWatcher отслеживает изменения в конфигурации и применяет их автоматически
type ConfigWatcher struct {
	configService *ConfigService
	taskService   *TaskService
	logger        *zap.Logger
	stopChan      chan struct{}
}

// NewConfigWatcher создает новый наблюдатель конфигурации
func NewConfigWatcher(configService *ConfigService, taskService *TaskService, logger *zap.Logger) *ConfigWatcher {
	return &ConfigWatcher{
		configService: configService,
		taskService:   taskService,
		logger:        logger,
		stopChan:      make(chan struct{}),
	}
}

// Start запускает наблюдение за изменениями конфигурации
func (w *ConfigWatcher) Start(ctx context.Context) {
	w.logger.Info("Starting config watcher")

	ticker := time.NewTicker(30 * time.Second) // Проверяем каждые 30 секунд
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Config watcher stopped due to context cancellation")
			return
		case <-w.stopChan:
			w.logger.Info("Config watcher stopped")
			return
		case <-ticker.C:
			w.checkForConfigChanges(ctx)
		}
	}
}

// Stop останавливает наблюдение за изменениями конфигурации
func (w *ConfigWatcher) Stop() {
	close(w.stopChan)
}

// checkForConfigChanges проверяет изменения в конфигурации
func (w *ConfigWatcher) checkForConfigChanges(ctx context.Context) {
	// Получаем все конфигурации
	configs, err := w.configService.GetAll()
	if err != nil {
		w.logger.Error("Failed to get configs for watching", zap.Error(err))
		return
	}

	// Здесь можно добавить логику для применения изменений
	// Например, перезагрузка настроек скрейпера, изменение интервалов задач и т.д.
	w.logger.Debug("Config watcher checked for changes", zap.String("configs", configs))
}

// ApplyConfigChanges применяет изменения конфигурации
func (w *ConfigWatcher) ApplyConfigChanges(configs []model.Config) error {
	for _, config := range configs {
		switch config.Key {
		case "SCRAPER_DELAY":
			// Применяем изменения задержки скрейпера
			w.logger.Info("Applying scraper delay change", zap.String("value", config.Value))
		case "SCRAPER_TIMEOUT":
			// Применяем изменения таймаута скрейпера
			w.logger.Info("Applying scraper timeout change", zap.String("value", config.Value))
		case "PLAYLIST_UPDATE_HOURS":
			// Применяем изменения интервала обновления плейлистов
			w.logger.Info("Applying playlist update hours change", zap.String("value", config.Value))
		case "LOG_LEVEL":
			// Применяем изменения уровня логирования
			w.logger.Info("Applying log level change", zap.String("value", config.Value))
		case "HOMEWORK_RESET_TIME":
			// Обновляем cron выражение для задачи сброса домашних заданий
			w.logger.Info("Applying homework reset time change", zap.String("value", config.Value))
			err := w.taskService.UpdateHomeworkResetCron(w.configService)
			if err != nil {
				w.logger.Error("Failed to update homework reset cron", zap.Error(err))
			}
		default:
			w.logger.Debug("No specific handler for config key", zap.String("key", config.Key))
		}
	}
	return nil
}
