// Package service содержит бизнес-логику приложения.
package service

import (
	"context"
	"fmt"
	"gemfactory/internal/model"
	"gemfactory/internal/storage/repository"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// TaskService содержит бизнес-логику для работы с задачами
type TaskService struct {
	repo   model.TaskRepository
	logger *zap.Logger
}

// NewTaskService создает новый сервис задач
func NewTaskService(db *bun.DB, logger *zap.Logger) *TaskService {
	return &TaskService{
		repo:   repository.NewTaskRepository(db, logger),
		logger: logger,
	}
}

// GetAllTasks возвращает все задачи
func (s *TaskService) GetAllTasks() ([]model.Task, error) {
	return s.repo.GetAll()
}

// GetActiveTasks возвращает активные задачи
func (s *TaskService) GetActiveTasks() ([]model.Task, error) {
	return s.repo.GetActive()
}

// GetDueTasks возвращает задачи, которые нужно выполнить
func (s *TaskService) GetDueTasks() ([]model.Task, error) {
	return s.repo.GetDueTasks()
}

// CreateTask создает новую задачу
func (s *TaskService) CreateTask(task *model.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("task validation failed: %w", err)
	}

	return s.repo.Create(task)
}

// UpdateTask обновляет задачу
func (s *TaskService) UpdateTask(task *model.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("task validation failed: %w", err)
	}

	return s.repo.Update(task)
}

// DeleteTask удаляет задачу
func (s *TaskService) DeleteTask(taskID int) error {
	return s.repo.Delete(taskID)
}

// UpdateRunStats обновляет статистику выполнения задачи
func (s *TaskService) UpdateRunStats(taskID int, success bool, err error) error {
	return s.repo.UpdateRunStats(taskID, success, err)
}

// GetTasksByType возвращает задачи по типу
func (s *TaskService) GetTasksByType(taskType model.TaskType) ([]model.Task, error) {
	return s.repo.GetByType(taskType)
}

// GetParseReleaseTasks возвращает задачи парсинга релизов
func (s *TaskService) GetParseReleaseTasks() ([]model.Task, error) {
	return s.GetTasksByType(model.TaskTypeParseReleases)
}

// ExecuteTask выполняет задачу
func (s *TaskService) ExecuteTask(ctx context.Context, task *model.Task, executor TaskExecutor) error {
	s.logger.Info("Executing task",
		zap.String("task_name", task.Name),
		zap.String("task_type", task.TaskType.String()))

	startTime := time.Now()
	err := executor.Execute(ctx, task)
	duration := time.Since(startTime)

	success := err == nil
	updateErr := s.UpdateRunStats(task.TaskID, success, err)
	if updateErr != nil {
		s.logger.Error("Failed to update task run stats",
			zap.String("task_name", task.Name),
			zap.Error(updateErr))
	}

	if success {
		s.logger.Info("Task executed successfully",
			zap.String("task_name", task.Name),
			zap.Duration("duration", duration))
	} else {
		s.logger.Error("Task execution failed",
			zap.String("task_name", task.Name),
			zap.Duration("duration", duration),
			zap.Error(err))
	}

	return err
}

// TaskExecutor определяет интерфейс для выполнения задач
type TaskExecutor interface {
	Execute(ctx context.Context, task *model.Task) error
}

// ParseReleaseTaskExecutor выполняет задачи парсинга релизов
type ParseReleaseTaskExecutor struct {
	releaseService *ReleaseService
	logger         *zap.Logger
}

// NewParseReleaseTaskExecutor создает новый исполнитель задач парсинга релизов
func NewParseReleaseTaskExecutor(releaseService *ReleaseService, logger *zap.Logger) *ParseReleaseTaskExecutor {
	return &ParseReleaseTaskExecutor{
		releaseService: releaseService,
		logger:         logger,
	}
}

// Execute выполняет задачу парсинга релизов
func (e *ParseReleaseTaskExecutor) Execute(ctx context.Context, task *model.Task) error {
	months, err := e.getMonthsToParse(task)
	if err != nil {
		return fmt.Errorf("failed to get months to parse: %w", err)
	}

	totalSaved := 0
	for _, month := range months {
		e.logger.Info("Parsing releases for month",
			zap.String("task_name", task.Name),
			zap.String("month", month))

		count, err := e.releaseService.ParseReleasesForMonth(ctx, month)
		if err != nil {
			e.logger.Error("Failed to parse releases for month",
				zap.String("month", month),
				zap.Error(err))
			continue
		}

		totalSaved += count
		e.logger.Info("Parsed releases for month",
			zap.String("month", month),
			zap.Int("count", count))
	}

	e.logger.Info("Task completed",
		zap.String("task_name", task.Name),
		zap.Int("total_saved", totalSaved))

	return nil
}

// getMonthsToParse определяет, какие месяцы нужно парсить
func (e *ParseReleaseTaskExecutor) getMonthsToParse(task *model.Task) ([]string, error) {
	monthsConfig, exists := task.GetConfigString("months")
	if !exists {
		return nil, fmt.Errorf("months configuration not found in task config")
	}

	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	switch monthsConfig {
	case "current+2":
		// Текущий месяц + 2 следующих
		return e.getCurrentAndNextMonths(currentMonth, currentYear, 2), nil
	case "previous_current_year":
		// Предыдущие месяцы текущего года
		return e.getPreviousMonthsOfCurrentYear(currentMonth, currentYear), nil
	default:
		return nil, fmt.Errorf("unknown months configuration: %s", monthsConfig)
	}
}

// getCurrentAndNextMonths возвращает текущий месяц и следующие N месяцев
func (e *ParseReleaseTaskExecutor) getCurrentAndNextMonths(currentMonth, currentYear, nextCount int) []string {
	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	var result []string
	for i := 0; i <= nextCount; i++ {
		monthIndex := (currentMonth - 1 + i) % 12
		if monthIndex < 0 {
			monthIndex += 12
		}
		// Формируем строку с годом
		monthWithYear := fmt.Sprintf("%s-%d", months[monthIndex], currentYear)
		result = append(result, monthWithYear)
	}

	return result
}

// getPreviousMonthsOfCurrentYear возвращает предыдущие месяцы текущего года
func (e *ParseReleaseTaskExecutor) getPreviousMonthsOfCurrentYear(currentMonth, currentYear int) []string {
	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	var result []string
	for i := 1; i < currentMonth; i++ {
		// Формируем строку с годом
		monthWithYear := fmt.Sprintf("%s-%d", months[i-1], currentYear)
		result = append(result, monthWithYear)
	}

	return result
}

// HomeworkResetTaskExecutor выполняет задачи сброса домашних заданий
type HomeworkResetTaskExecutor struct {
	homeworkService *HomeworkService
	configService   *ConfigService
	logger          *zap.Logger
}

// NewHomeworkResetTaskExecutor создает новый исполнитель задач сброса домашних заданий
func NewHomeworkResetTaskExecutor(homeworkService *HomeworkService, configService *ConfigService, logger *zap.Logger) *HomeworkResetTaskExecutor {
	return &HomeworkResetTaskExecutor{
		homeworkService: homeworkService,
		configService:   configService,
		logger:          logger,
	}
}

// Execute выполняет задачу сброса домашних заданий
func (e *HomeworkResetTaskExecutor) Execute(ctx context.Context, task *model.Task) error {
	e.logger.Info("Starting homework reset task")

	// Сбрасываем все домашние задания
	err := e.homeworkService.ResetAllHomework()
	if err != nil {
		return fmt.Errorf("failed to reset homework: %w", err)
	}

	e.logger.Info("Homework reset task completed successfully")
	return nil
}

// UpdateHomeworkResetCron обновляет cron выражение для задачи сброса домашних заданий на основе конфигурации
func (s *TaskService) UpdateHomeworkResetCron(configService *ConfigService) error {
	// Получаем время сброса из конфигурации
	resetTime, err := configService.GetConfigValue("HOMEWORK_RESET_TIME")
	if err != nil {
		return fmt.Errorf("failed to get HOMEWORK_RESET_TIME from config: %w", err)
	}

	if resetTime == "" {
		s.logger.Warn("HOMEWORK_RESET_TIME not configured, using default 00:00")
		resetTime = "00:00"
	}

	// Парсим время (формат HH:MM)
	timeParts := strings.Split(resetTime, ":")
	if len(timeParts) != 2 {
		return fmt.Errorf("invalid time format: %s, expected HH:MM", resetTime)
	}

	hour := timeParts[0]
	minute := timeParts[1]

	// Формируем cron выражение (каждый день в указанное время)
	cronExpression := fmt.Sprintf("%s %s * * *", minute, hour)

	// Получаем задачу homework_reset
	tasks, err := s.repo.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	var homeworkResetTask *model.Task
	for _, task := range tasks {
		if task.Name == "homework_reset" {
			homeworkResetTask = &task
			break
		}
	}

	if homeworkResetTask == nil {
		return fmt.Errorf("homework_reset task not found")
	}

	// Обновляем cron выражение если оно изменилось
	if homeworkResetTask.CronExpression != cronExpression {
		homeworkResetTask.CronExpression = cronExpression
		err = s.repo.Update(homeworkResetTask)
		if err != nil {
			return fmt.Errorf("failed to update homework_reset task cron: %w", err)
		}

		s.logger.Info("Updated homework_reset task cron expression",
			zap.String("old_cron", homeworkResetTask.CronExpression),
			zap.String("new_cron", cronExpression),
			zap.String("reset_time", resetTime))
	}

	return nil
}
