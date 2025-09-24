// Package repository содержит репозитории для работы с базой данных.
package repository

import (
	"context"
	"fmt"
	"gemfactory/internal/model"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ReleaseTypeRepository реализует интерфейс для работы с типами релизов
type ReleaseTypeRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewReleaseTypeRepository создает новый репозиторий типов релизов
func NewReleaseTypeRepository(db *bun.DB, logger *zap.Logger) *ReleaseTypeRepository {
	return &ReleaseTypeRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID возвращает тип релиза по ID
func (r *ReleaseTypeRepository) GetByID(id int) (*model.ReleaseTypeModel, error) {
	ctx := context.Background()
	releaseType := new(model.ReleaseTypeModel)

	err := r.db.NewSelect().
		Model(releaseType).
		Where("release_type_id = ?", id).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query release type by ID: %w", err)
	}

	return releaseType, nil
}

// GetAll возвращает все типы релизов
func (r *ReleaseTypeRepository) GetAll() ([]model.ReleaseTypeModel, error) {
	ctx := context.Background()
	var releaseTypes []model.ReleaseTypeModel

	err := r.db.NewSelect().
		Model(&releaseTypes).
		Order("name ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query release types: %w", err)
	}

	return releaseTypes, nil
}

// GetByName возвращает тип релиза по имени
func (r *ReleaseTypeRepository) GetByName(name string) (*model.ReleaseTypeModel, error) {
	ctx := context.Background()
	releaseType := new(model.ReleaseTypeModel)

	err := r.db.NewSelect().
		Model(releaseType).
		Where("name = ?", name).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query release type by name: %w", err)
	}

	return releaseType, nil
}

// Create создает новый тип релиза
func (r *ReleaseTypeRepository) Create(releaseType *model.ReleaseTypeModel) error {
	ctx := context.Background()

	_, err := r.db.NewInsert().
		Model(releaseType).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create release type: %w", err)
	}

	return nil
}

// Update обновляет тип релиза
func (r *ReleaseTypeRepository) Update(releaseType *model.ReleaseTypeModel) error {
	ctx := context.Background()

	_, err := r.db.NewUpdate().
		Model(releaseType).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update release type: %w", err)
	}

	return nil
}

// Delete удаляет тип релиза
func (r *ReleaseTypeRepository) Delete(id int) error {
	ctx := context.Background()

	_, err := r.db.NewDelete().
		Model((*model.ReleaseTypeModel)(nil)).
		Where("release_type_id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete release type: %w", err)
	}

	return nil
}
