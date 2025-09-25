// Package repository содержит репозитории для работы с базой данных.
package repository

import (
	"context"
	"fmt"
	"time"

	"gemfactory/internal/model"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// HomeworkTrackingRepository реализует интерфейс для работы с отслеживанием домашних заданий
type HomeworkTrackingRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewHomeworkTrackingRepository создает новый репозиторий для отслеживания домашних заданий
func NewHomeworkTrackingRepository(db *bun.DB, logger *zap.Logger) *HomeworkTrackingRepository {
	return &HomeworkTrackingRepository{
		db:     db,
		logger: logger,
	}
}

// GetByUserID возвращает все домашние задания пользователя
func (r *HomeworkTrackingRepository) GetByUserID(userID int64) ([]model.HomeworkTracking, error) {
	ctx := context.Background()
	var trackings []model.HomeworkTracking

	err := r.db.NewSelect().
		Model(&trackings).
		Where("user_id = ?", userID).
		Order("issued_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get homework tracking by user_id: %w", err)
	}

	return trackings, nil
}

// GetCompletedByUserID возвращает завершенные домашние задания пользователя
func (r *HomeworkTrackingRepository) GetCompletedByUserID(userID int64) ([]model.HomeworkTracking, error) {
	ctx := context.Background()
	var trackings []model.HomeworkTracking

	err := r.db.NewSelect().
		Model(&trackings).
		Where("user_id = ? AND is_completed = true", userID).
		Order("completed_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get completed homework tracking by user_id: %w", err)
	}

	return trackings, nil
}

// GetPendingByUserID возвращает незавершенные домашние задания пользователя
func (r *HomeworkTrackingRepository) GetPendingByUserID(userID int64) ([]model.HomeworkTracking, error) {
	ctx := context.Background()
	var trackings []model.HomeworkTracking

	err := r.db.NewSelect().
		Model(&trackings).
		Where("user_id = ? AND is_completed = false", userID).
		Order("issued_at ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get pending homework tracking by user_id: %w", err)
	}

	return trackings, nil
}

// Create создает новое отслеживание домашнего задания
func (r *HomeworkTrackingRepository) Create(tracking *model.HomeworkTracking) error {
	ctx := context.Background()

	_, err := r.db.NewInsert().
		Model(tracking).
		On("CONFLICT (user_id, track_id, spotify_id) DO UPDATE").
		Set("issued_at = EXCLUDED.issued_at").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create homework tracking: %w", err)
	}

	return nil
}

// Update обновляет отслеживание домашнего задания
func (r *HomeworkTrackingRepository) Update(tracking *model.HomeworkTracking) error {
	ctx := context.Background()

	_, err := r.db.NewUpdate().
		Model(tracking).
		Where("id = ?", tracking.ID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update homework tracking: %w", err)
	}

	return nil
}

// MarkCompleted отмечает домашнее задание как завершенное
func (r *HomeworkTrackingRepository) MarkCompleted(userID int64, trackID string, spotifyID string) error {
	ctx := context.Background()
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*model.HomeworkTracking)(nil)).
		Set("is_completed = true").
		Set("completed_at = ?", now).
		Set("updated_at = ?", now).
		Where("user_id = ? AND track_id = ? AND spotify_id = ?", userID, trackID, spotifyID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to mark homework as completed: %w", err)
	}

	return nil
}

// GetIssuedTrackIDs возвращает список track_id уже выданных треков пользователю для конкретного плейлиста
func (r *HomeworkTrackingRepository) GetIssuedTrackIDs(userID int64, spotifyID string) ([]string, error) {
	ctx := context.Background()
	var trackIDs []string

	err := r.db.NewSelect().
		Model((*model.HomeworkTracking)(nil)).
		Column("track_id").
		Where("user_id = ? AND spotify_id = ?", userID, spotifyID).
		Scan(ctx, &trackIDs)

	if err != nil {
		return nil, fmt.Errorf("failed to get issued track IDs: %w", err)
	}

	return trackIDs, nil
}

// CanRequestHomework проверяет может ли пользователь запросить новое домашнее задание
// Теперь эта логика перенесена в HomeworkService, который имеет доступ к конфигурации
func (r *HomeworkTrackingRepository) CanRequestHomework(userID int64) (bool, error) {
	lastTime, err := r.GetLastRequestTime(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get last request time: %w", err)
	}

	// Если домашних заданий не было, можно запросить
	if lastTime == nil {
		return true, nil
	}

	timeSinceLastRequest := time.Since(*lastTime)
	return timeSinceLastRequest >= time.Hour, nil
}

// GetLastRequestTime возвращает время последнего запроса
func (r *HomeworkTrackingRepository) GetLastRequestTime(userID int64) (*time.Time, error) {
	ctx := context.Background()
	var lastTime time.Time

	err := r.db.NewSelect().
		Model((*model.HomeworkTracking)(nil)).
		Column("issued_at").
		Where("user_id = ?", userID).
		Order("issued_at DESC").
		Limit(1).
		Scan(ctx, &lastTime)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last request time: %w", err)
	}

	return &lastTime, nil
}

// GetAllPending возвращает все незавершенные домашние задания всех пользователей
func (r *HomeworkTrackingRepository) GetAllPending() ([]model.HomeworkTracking, error) {
	ctx := context.Background()

	var trackings []model.HomeworkTracking
	err := r.db.NewSelect().
		Model(&trackings).
		Where("is_completed = ?", false).
		Order("issued_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all pending homework: %w", err)
	}
	return trackings, nil
}
