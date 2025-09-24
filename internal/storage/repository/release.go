// Package repository содержит репозитории для работы с базой данных.
package repository

import (
	"context"
	"fmt"
	"gemfactory/internal/model"
	"sort"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ReleaseRepository реализует интерфейс для работы с релизами
type ReleaseRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewReleaseRepository создает новый репозиторий релизов
func NewReleaseRepository(db *bun.DB, logger *zap.Logger) *ReleaseRepository {
	return &ReleaseRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID возвращает релиз по ID
func (r *ReleaseRepository) GetByID(id int) (*model.Release, error) {
	ctx := context.Background()
	release := new(model.Release)

	err := r.db.NewSelect().
		Model(release).
		Relation("Artist").
		Where("release_id = ?", id).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query release by ID: %w", err)
	}

	return release, nil
}

// GetAll возвращает все релизы
func (r *ReleaseRepository) GetAll() ([]model.Release, error) {
	ctx := context.Background()
	var releases []model.Release

	err := r.db.NewSelect().
		Model(&releases).
		Order("date ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query releases: %w", err)
	}

	return releases, nil
}

// GetByArtistAndTitle возвращает релиз по артисту и названию
func (r *ReleaseRepository) GetByArtistAndTitle(artistID int, title string) (*model.Release, error) {
	ctx := context.Background()
	var release model.Release

	err := r.db.NewSelect().
		Model(&release).
		Where("artist_id = ? AND title = ?", artistID, title).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // Релиз не найден
		}
		return nil, fmt.Errorf("failed to query release by artist and title: %w", err)
	}

	return &release, nil
}

// GetByArtistDateAndTrack возвращает релиз по артисту, дате и треку
func (r *ReleaseRepository) GetByArtistDateAndTrack(artistID int, date, titleTrack string) (*model.Release, error) {
	ctx := context.Background()
	var release model.Release

	err := r.db.NewSelect().
		Model(&release).
		Where("artist_id = ? AND date = ? AND title_track = ?", artistID, date, titleTrack).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // Релиз не найден
		}
		return nil, fmt.Errorf("failed to query release by artist, date and track: %w", err)
	}

	return &release, nil
}

// GetByGender возвращает релизы по полу
func (r *ReleaseRepository) GetByGender(gender model.Gender) ([]model.Release, error) {
	ctx := context.Background()
	var releases []model.Release

	err := r.db.NewSelect().
		Model(&releases).
		Relation("Artist").
		Where("artist.gender = ?", gender).
		Where("artist.is_active = ?", true).
		Order("release.date ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query releases by gender: %w", err)
	}

	return releases, nil
}

// Create создает новый релиз
func (r *ReleaseRepository) Create(release *model.Release) error {
	ctx := context.Background()

	_, err := r.db.NewInsert().
		Model(release).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	return nil
}

// Update обновляет релиз
func (r *ReleaseRepository) Update(release *model.Release) error {
	ctx := context.Background()

	_, err := r.db.NewUpdate().
		Model(release).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update release: %w", err)
	}

	return nil
}

// Delete удаляет релиз
func (r *ReleaseRepository) Delete(id int) error {
	ctx := context.Background()

	_, err := r.db.NewDelete().
		Model((*model.Release)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}

	return nil
}

// GetByArtist возвращает релизы по артисту
func (r *ReleaseRepository) GetByArtist(artistID int) ([]model.Release, error) {
	ctx := context.Background()
	var releases []model.Release

	err := r.db.NewSelect().
		Model(&releases).
		Relation("Artist").
		Where("artist_id = ?", artistID).
		Order("date ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query releases by artist: %w", err)
	}

	return releases, nil
}

// GetByArtistName возвращает релизы по имени артиста (без учета is_active)
func (r *ReleaseRepository) GetByArtistName(artistName string) ([]model.Release, error) {
	ctx := context.Background()
	var releases []model.Release

	err := r.db.NewSelect().
		Model(&releases).
		Relation("Artist").
		Where("LOWER(artist.name) = LOWER(?)", artistName).
		Order("date ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query releases by artist name: %w", err)
	}

	// Сортируем релизы по дате в Go коде
	sort.Slice(releases, func(i, j int) bool {
		// Парсим даты в формате DD.MM.YY
		dateI, errI := time.Parse("02.01.06", releases[i].Date)
		dateJ, errJ := time.Parse("02.01.06", releases[j].Date)

		// Если не удалось распарсить, используем строковую сортировку
		if errI != nil || errJ != nil {
			return releases[i].Date < releases[j].Date
		}

		return dateI.Before(dateJ)
	})

	return releases, nil
}

// GetByDateRange возвращает релизы в диапазоне дат
func (r *ReleaseRepository) GetByDateRange(start, end time.Time) ([]model.Release, error) {
	ctx := context.Background()
	var releases []model.Release

	err := r.db.NewSelect().
		Model(&releases).
		Relation("Artist").
		Where("release.date >= ? AND release.date <= ?", start, end).
		Order("release.date ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query releases by date range: %w", err)
	}

	return releases, nil
}

// GetActive возвращает активные релизы
func (r *ReleaseRepository) GetActive() ([]model.Release, error) {
	ctx := context.Background()
	var releases []model.Release

	err := r.db.NewSelect().
		Model(&releases).
		Relation("Artist").
		Where("release.is_active = ?", true).
		Order("release.date ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query active releases: %w", err)
	}

	return releases, nil
}

// GetWithRelations возвращает релизы с загруженными связями
func (r *ReleaseRepository) GetWithRelations() ([]model.Release, error) {
	ctx := context.Background()
	var releases []model.Release

	err := r.db.NewSelect().
		Model(&releases).
		Relation("Artist").
		Order("date ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query releases with relations: %w", err)
	}

	return releases, nil
}
