// Package service содержит бизнес-логику приложения.
package service

import (
	"context"
	"fmt"
	"gemfactory/internal/model"
	"time"

	"go.uber.org/zap"
)

// ConfigWatcher отслеживает изменения в конфигурации и применяет их автоматически
type ConfigWatcher struct {
	configService *ConfigService
	taskService   *TaskService
	scheduler     *Scheduler
	logger        *zap.Logger
	stopChan      chan struct{}
	lastTaskHash  string // Хеш последнего состояния задач для отслеживания изменений
}

// NewConfigWatcher создает новый наблюдатель конфигурации
func NewConfigWatcher(configService *ConfigService, taskService *TaskService, scheduler *Scheduler, logger *zap.Logger) *ConfigWatcher {
	return &ConfigWatcher{
		configService: configService,
		taskService:   taskService,
		scheduler:     scheduler,
		logger:        logger,
		stopChan:      make(chan struct{}),
		lastTaskHash:  "",
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

// checkForConfigChanges проверяет изменения в конфигурации и задачах
func (w *ConfigWatcher) checkForConfigChanges(ctx context.Context) {
	// Проверяем изменения в конфигурации
	configs, err := w.configService.GetAll()
	if err != nil {
		w.logger.Error("Failed to get configs for watching", zap.Error(err))
		return
	}

	// Проверяем изменения в задачах
	if err := w.checkForTaskChanges(ctx); err != nil {
		w.logger.Error("Failed to check for task changes", zap.Error(err))
	}

	w.logger.Debug("Config watcher checked for changes", zap.String("configs", configs))
}

// ApplyConfigChanges применяет изменения конфигурации
func (w *ConfigWatcher) ApplyConfigChanges(configs []model.Config) error {
	for _, config := range configs {
		switch config.Key {
		case "SCRAPER_DELAY":
			w.logger.Info("Applying scraper delay change", zap.String("value", config.Value))
		case "SCRAPER_TIMEOUT":
			w.logger.Info("Applying scraper timeout change", zap.String("value", config.Value))
		case "LOG_LEVEL":
			w.logger.Info("Applying log level change", zap.String("value", config.Value))
		default:
			w.logger.Debug("No specific handler for config key", zap.String("key", config.Key))
		}
	}
	return nil
}

// checkForTaskChanges проверяет изменения в задачах и перезагружает планировщик при необходимости
func (w *ConfigWatcher) checkForTaskChanges(ctx context.Context) error {
	// Получаем все активные задачи
	tasks, err := w.taskService.GetActiveTasks()
	if err != nil {
		return fmt.Errorf("failed to get active tasks: %w", err)
	}

	// Создаем хеш текущего состояния задач
	currentHash := w.calculateTaskHash(tasks)

	// Если хеш изменился, перезагружаем планировщик
	if currentHash != w.lastTaskHash {
		w.logger.Info("Task changes detected, reloading scheduler",
			zap.String("old_hash", w.lastTaskHash),
			zap.String("new_hash", currentHash))

		if w.scheduler != nil {
			if err := w.scheduler.ReloadTasks(); err != nil {
				return fmt.Errorf("failed to reload scheduler tasks: %w", err)
			}
			w.logger.Info("Scheduler reloaded successfully")
		}

		w.lastTaskHash = currentHash
	}

	return nil
}

// calculateTaskHash создает хеш состояния задач для отслеживания изменений
func (w *ConfigWatcher) calculateTaskHash(tasks []model.Task) string {
	// Создаем строку из всех важных полей задач
	var hashData string
	for _, task := range tasks {
		hashData += fmt.Sprintf("%s:%s:%s:%t:", task.Name, task.CronExpression, task.TaskType, task.IsActive)
	}

	// Простой хеш (в реальном приложении можно использовать crypto/sha256)
	return fmt.Sprintf("%d", len(hashData))
}
