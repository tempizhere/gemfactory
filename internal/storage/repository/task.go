// Package repository содержит реализации репозиториев для работы с базой данных.
package repository

import (
	"context"
	"fmt"
	"gemfactory/internal/model"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// TaskRepository реализует интерфейс model.TaskRepository
type TaskRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewTaskRepository создает новый репозиторий задач
func NewTaskRepository(db *bun.DB, logger *zap.Logger) model.TaskRepository {
	return &TaskRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID получает задачу по ID
func (r *TaskRepository) GetByID(id int) (*model.Task, error) {
	var task model.Task
	ctx := context.Background()
	err := r.db.NewSelect().Model(&task).Where("task_id = ?", id).Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get task by ID: %w", err)
	}
	return &task, nil
}

// GetAll получает все задачи
func (r *TaskRepository) GetAll() ([]model.Task, error) {
	var tasks []model.Task
	ctx := context.Background()
	err := r.db.NewSelect().Model(&tasks).Order("name").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tasks: %w", err)
	}
	return tasks, nil
}

// Create создает новую задачу
func (r *TaskRepository) Create(task *model.Task) error {
	ctx := context.Background()
	_, err := r.db.NewInsert().Model(task).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

// Update обновляет задачу
func (r *TaskRepository) Update(task *model.Task) error {
	ctx := context.Background()
	_, err := r.db.NewUpdate().Model(task).Where("task_id = ?", task.TaskID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}

// Delete удаляет задачу
func (r *TaskRepository) Delete(id int) error {
	ctx := context.Background()
	_, err := r.db.NewDelete().Model((*model.Task)(nil)).Where("task_id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// GetByType получает задачи по типу
func (r *TaskRepository) GetByType(taskType model.TaskType) ([]model.Task, error) {
	var tasks []model.Task
	ctx := context.Background()
	err := r.db.NewSelect().Model(&tasks).Where("task_type = ?", taskType).Order("name").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by type: %w", err)
	}
	return tasks, nil
}

// GetActive получает активные задачи
func (r *TaskRepository) GetActive() ([]model.Task, error) {
	var tasks []model.Task
	ctx := context.Background()
	err := r.db.NewSelect().Model(&tasks).Where("is_active = ?", true).Order("name").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks: %w", err)
	}
	return tasks, nil
}

// GetDueTasks получает задачи, которые нужно выполнить
func (r *TaskRepository) GetDueTasks() ([]model.Task, error) {
	var tasks []model.Task
	ctx := context.Background()
	err := r.db.NewSelect().Model(&tasks).
		Where("is_active = ? AND (next_run IS NULL OR next_run <= ?)", true, "NOW()").
		Order("next_run ASC NULLS FIRST").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get due tasks: %w", err)
	}
	return tasks, nil
}

// UpdateRunStats обновляет статистику выполнения задачи
func (r *TaskRepository) UpdateRunStats(taskID int, success bool, execErr error) error {
	ctx := context.Background()

	// Сначала получаем задачу для вычисления next_run
	task, err := r.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task for next_run calculation: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task with ID %d not found", taskID)
	}

	// Вычисляем следующее время запуска
	nextRun, err := r.calculateNextRun(task.CronExpression)
	if err != nil {
		return fmt.Errorf("failed to calculate next run: %w", err)
	}

	var lastError string
	if execErr != nil {
		lastError = execErr.Error()
	}

	query := r.db.NewUpdate().Model((*model.Task)(nil)).
		Set("run_count = run_count + 1").
		Set("last_run = NOW()").
		Set("next_run = ?", nextRun).
		Set("last_error = ?", lastError).
		Where("task_id = ?", taskID)

	if success {
		query = query.Set("success_count = success_count + 1")
	} else {
		query = query.Set("error_count = error_count + 1")
	}

	_, updateErr := query.Exec(ctx)
	if updateErr != nil {
		return fmt.Errorf("failed to update task run stats: %w", updateErr)
	}

	return nil
}

// calculateNextRun вычисляет следующее время запуска задачи на основе cron выражения
func (r *TaskRepository) calculateNextRun(cronExpression string) (time.Time, error) {
	// Парсим cron выражение
	schedule, err := cron.ParseStandard(cronExpression)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse cron expression %s: %w", cronExpression, err)
	}

	// Вычисляем следующее время запуска от текущего времени
	nextRun := schedule.Next(time.Now())
	return nextRun, nil
}

// GetByName получает задачу по имени
func (r *TaskRepository) GetByName(name string) (*model.Task, error) {
	ctx := context.Background()
	var task model.Task

	err := r.db.NewSelect().
		Model(&task).
		Where("name = ?", name).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get task by name %s: %w", name, err)
	}

	return &task, nil
}
