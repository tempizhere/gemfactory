// Package repository содержит репозитории для работы с базой данных.
package repository

import (
	"context"
	"fmt"
	"gemfactory/internal/model"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// HomeworkRepository реализует интерфейс для работы с домашними заданиями
type HomeworkRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewHomeworkRepository создает новый репозиторий домашних заданий
func NewHomeworkRepository(db *bun.DB, logger *zap.Logger) *HomeworkRepository {
	return &HomeworkRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID возвращает домашнее задание по ID
func (r *HomeworkRepository) GetByID(id int) (*model.Homework, error) {
	ctx := context.Background()
	homework := new(model.Homework)

	err := r.db.NewSelect().
		Model(homework).
		Where("homework_id = ?", id).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query homework by ID: %w", err)
	}

	return homework, nil
}

// GetAll возвращает все домашние задания
func (r *HomeworkRepository) GetAll() ([]model.Homework, error) {
	ctx := context.Background()
	var homework []model.Homework

	err := r.db.NewSelect().
		Model(&homework).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query homework: %w", err)
	}

	return homework, nil
}

// GetByUserID возвращает домашние задания пользователя
func (r *HomeworkRepository) GetByUserID(userID int64) ([]model.Homework, error) {
	ctx := context.Background()
	var homework []model.Homework

	err := r.db.NewSelect().
		Model(&homework).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query homework: %w", err)
	}

	return homework, nil
}

// GetActiveByUserID возвращает активное домашнее задание пользователя
func (r *HomeworkRepository) GetActiveByUserID(userID int64) (*model.Homework, error) {
	ctx := context.Background()
	homework := new(model.Homework)

	err := r.db.NewSelect().
		Model(homework).
		Where("user_id = ? AND completed = false", userID).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan homework: %w", err)
	}

	return homework, nil
}

// Create создает новое домашнее задание
func (r *HomeworkRepository) Create(homework *model.Homework) error {
	ctx := context.Background()

	_, err := r.db.NewInsert().
		Model(homework).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create homework: %w", err)
	}

	return nil
}

// Update обновляет домашнее задание
func (r *HomeworkRepository) Update(homework *model.Homework) error {
	ctx := context.Background()

	_, err := r.db.NewUpdate().
		Model(homework).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update homework: %w", err)
	}

	return nil
}

// Delete удаляет домашнее задание
func (r *HomeworkRepository) Delete(id int) error {
	ctx := context.Background()

	_, err := r.db.NewDelete().
		Model((*model.Homework)(nil)).
		Where("homework_id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete homework: %w", err)
	}

	return nil
}

// MarkCompleted отмечает домашнее задание как выполненное
func (r *HomeworkRepository) MarkCompleted(id int) error {
	ctx := context.Background()

	_, err := r.db.NewUpdate().
		Model((*model.Homework)(nil)).
		Set("completed = true").
		Set("updated_at = NOW()").
		Where("homework_id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to mark homework as completed: %w", err)
	}

	return nil
}

// GetRandomTrack возвращает случайный трек для домашнего задания
func (r *HomeworkRepository) GetRandomTrack() (*model.Homework, error) {
	ctx := context.Background()
	homework := new(model.Homework)

	err := r.db.NewSelect().
		Model(homework).
		OrderExpr("RANDOM()").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan homework: %w", err)
	}

	return homework, nil
}

// CanRequestHomework проверяет, может ли пользователь запросить домашнее задание
func (r *HomeworkRepository) CanRequestHomework(userID int64) (bool, error) {
	ctx := context.Background()

	count, err := r.db.NewSelect().
		Model((*model.Homework)(nil)).
		Where("user_id = ? AND completed = false", userID).
		Count(ctx)

	if err != nil {
		return false, fmt.Errorf("failed to check homework count: %w", err)
	}

	return count == 0, nil
}

// GetLastRequestTime возвращает время последнего запроса
func (r *HomeworkRepository) GetLastRequestTime(userID int64) (*time.Time, error) {
	ctx := context.Background()
	var lastRequest time.Time

	err := r.db.NewSelect().
		Model((*model.Homework)(nil)).
		Column("created_at").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx, &lastRequest)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan last request time: %w", err)
	}

	return &lastRequest, nil
}
