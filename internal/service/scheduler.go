// Package service содержит планировщик задач.
package service

import (
	"context"
	"fmt"
	"gemfactory/internal/model"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler управляет выполнением задач по расписанию
type Scheduler struct {
	taskService *TaskService
	executors   map[model.TaskType]TaskExecutor
	cron        *cron.Cron
	logger      *zap.Logger
	mu          sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewScheduler создает новый планировщик
func NewScheduler(taskService *TaskService, logger *zap.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		taskService: taskService,
		executors:   make(map[model.TaskType]TaskExecutor),
		cron:        cron.New(cron.WithLocation(time.UTC)),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// RegisterExecutor регистрирует исполнитель для типа задачи
func (s *Scheduler) RegisterExecutor(taskType model.TaskType, executor TaskExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executors[taskType] = executor
	s.logger.Info("Registered task executor", zap.String("task_type", taskType.String()))
}

// Start запускает планировщик
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.logger.Info("Starting scheduler")

	// Загружаем активные задачи и добавляем их в cron
	tasks, err := s.taskService.GetActiveTasks()
	if err != nil {
		return fmt.Errorf("failed to get active tasks: %w", err)
	}

	s.logger.Info("Loaded active tasks from database", zap.Int("count", len(tasks)))
	for _, task := range tasks {
		s.logger.Info("Processing task",
			zap.String("name", task.Name),
			zap.String("type", string(task.TaskType)),
			zap.String("cron", task.CronExpression))

		if err := s.addTaskToCron(&task); err != nil {
			s.logger.Error("Failed to add task to cron",
				zap.String("task_name", task.Name),
				zap.String("task_type", string(task.TaskType)),
				zap.Error(err))
		}
	}

	// Запускаем cron
	s.cron.Start()
	s.running = true

	s.logger.Info("Scheduler started successfully", zap.Int("tasks_count", len(tasks)))

	// Запускаем горутину для проверки просроченных задач
	go s.runDueTasksChecker()

	return nil
}

// Stop останавливает планировщик
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.logger.Info("Stopping scheduler")

	s.cancel()
	s.cron.Stop()
	s.running = false

	s.logger.Info("Scheduler stopped")
}

// addTaskToCron добавляет задачу в cron
func (s *Scheduler) addTaskToCron(task *model.Task) error {
	executor, exists := s.executors[task.TaskType]
	if !exists {
		return fmt.Errorf("no executor registered for task type: %s", task.TaskType)
	}

	_, err := s.cron.AddFunc(task.CronExpression, func() {
		s.executeTask(task, executor)
	})

	if err != nil {
		return fmt.Errorf("failed to add task to cron: %w", err)
	}

	s.logger.Info("Added task to cron",
		zap.String("task_name", task.Name),
		zap.String("cron_expression", task.CronExpression))

	return nil
}

// executeTask выполняет задачу
func (s *Scheduler) executeTask(task *model.Task, executor TaskExecutor) {
	s.logger.Info("Executing scheduled task", zap.String("task_name", task.Name))

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	defer cancel()

	err := s.taskService.ExecuteTask(ctx, task, executor)
	if err != nil {
		s.logger.Error("Scheduled task execution failed",
			zap.String("task_name", task.Name),
			zap.Error(err))
	}
}

// runDueTasksChecker проверяет и выполняет просроченные задачи
func (s *Scheduler) runDueTasksChecker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndExecuteDueTasks()
		}
	}
}

// checkAndExecuteDueTasks проверяет и выполняет просроченные задачи
func (s *Scheduler) checkAndExecuteDueTasks() {
	tasks, err := s.taskService.GetDueTasks()
	if err != nil {
		s.logger.Error("Failed to get due tasks", zap.Error(err))
		return
	}

	if len(tasks) == 0 {
		return
	}

	s.logger.Info("Found due tasks", zap.Int("count", len(tasks)))

	for _, task := range tasks {
		executor, exists := s.executors[task.TaskType]
		if !exists {
			s.logger.Error("No executor for task type",
				zap.String("task_name", task.Name),
				zap.String("task_type", task.TaskType.String()))
			continue
		}

		go s.executeTask(&task, executor)
	}
}

// ReloadTasks перезагружает задачи из базы данных
func (s *Scheduler) ReloadTasks() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	s.logger.Info("Reloading tasks")

	// Останавливаем текущий cron
	s.cron.Stop()

	// Создаем новый cron
	s.cron = cron.New(cron.WithLocation(time.UTC))

	// Загружаем активные задачи
	tasks, err := s.taskService.GetActiveTasks()
	if err != nil {
		return fmt.Errorf("failed to get active tasks: %w", err)
	}

	// Добавляем задачи в новый cron
	for _, task := range tasks {
		if err := s.addTaskToCron(&task); err != nil {
			s.logger.Error("Failed to add task to cron",
				zap.String("task_name", task.Name),
				zap.Error(err))
		}
	}

	// Запускаем новый cron
	s.cron.Start()

	s.logger.Info("Tasks reloaded successfully", zap.Int("tasks_count", len(tasks)))
	return nil
}

// GetStatus возвращает статус планировщика
func (s *Scheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.cron.Entries()
	activeTasks := make([]map[string]interface{}, 0, len(entries))

	for _, entry := range entries {
		activeTasks = append(activeTasks, map[string]interface{}{
			"id":       entry.ID,
			"next_run": entry.Next,
		})
	}

	return map[string]interface{}{
		"running":      s.running,
		"active_tasks": len(activeTasks),
		"tasks":        activeTasks,
	}
}

// UpdatePlaylistTaskExecutor реализует TaskExecutor для обновления плейлистов
type UpdatePlaylistTaskExecutor struct {
	playlistService *PlaylistService
	configService   *ConfigService
	logger          *zap.Logger
}

// NewUpdatePlaylistTaskExecutor создает новый исполнитель задач обновления плейлистов
func NewUpdatePlaylistTaskExecutor(playlistService *PlaylistService, configService *ConfigService, logger *zap.Logger) *UpdatePlaylistTaskExecutor {
	return &UpdatePlaylistTaskExecutor{
		playlistService: playlistService,
		configService:   configService,
		logger:          logger,
	}
}

// Execute выполняет задачу обновления плейлиста
func (e *UpdatePlaylistTaskExecutor) Execute(ctx context.Context, task *model.Task) error {
	// Получаем URL плейлиста из конфигурации
	playlistURL, err := e.configService.GetConfigValue("PLAYLIST_URL")
	if err != nil {
		return fmt.Errorf("failed to get PLAYLIST_URL from config: %w", err)
	}

	if playlistURL == "" {
		return fmt.Errorf("PLAYLIST_URL is not configured")
	}

	// Обновляем плейлист
	err = e.playlistService.ReloadPlaylist()
	if err != nil {
		return fmt.Errorf("failed to update playlist: %w", err)
	}

	e.logger.Info("Playlist update task completed successfully",
		zap.String("task_name", task.Name),
		zap.String("playlist_url", playlistURL))

	return nil
}
