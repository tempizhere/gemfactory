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

// ArtistRepository реализует интерфейс для работы с артистами
type ArtistRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewArtistRepository создает новый репозиторий артистов
func NewArtistRepository(db *bun.DB, logger *zap.Logger) *ArtistRepository {
	return &ArtistRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID возвращает артиста по ID
func (r *ArtistRepository) GetByID(id int) (*model.Artist, error) {
	ctx := context.Background()
	artist := new(model.Artist)

	err := r.db.NewSelect().
		Model(artist).
		Where("artist_id = ?", id).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query artist by ID: %w", err)
	}

	return artist, nil
}

// GetAll возвращает всех артистов
func (r *ArtistRepository) GetAll() ([]model.Artist, error) {
	ctx := context.Background()
	var artists []model.Artist

	err := r.db.NewSelect().
		Model(&artists).
		Order("name ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query artists: %w", err)
	}

	return artists, nil
}

// GetByGender возвращает артистов по полу
func (r *ArtistRepository) GetByGender(gender model.Gender) ([]model.Artist, error) {
	ctx := context.Background()
	var artists []model.Artist

	err := r.db.NewSelect().
		Model(&artists).
		Where("gender = ?", gender).
		Order("name ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query artists by gender: %w", err)
	}

	return artists, nil
}

// GetByName возвращает артиста по имени (нечувствительно к регистру)
func (r *ArtistRepository) GetByName(name string) (*model.Artist, error) {
	ctx := context.Background()
	artist := new(model.Artist)

	// Нормализуем имя для поиска (приводим к нижнему регистру)
	normalizedName := strings.ToLower(strings.TrimSpace(name))

	err := r.db.NewSelect().
		Model(artist).
		Where("LOWER(name) = ?", normalizedName).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan artist: %w", err)
	}

	return artist, nil
}

// Create создает нового артиста
func (r *ArtistRepository) Create(artist *model.Artist) error {
	ctx := context.Background()

	// Сохраняем имя артиста точно как пришло с сайта
	// Никакой нормализации - только убираем лишние пробелы по краям
	artist.Name = strings.TrimSpace(artist.Name)

	_, err := r.db.NewInsert().
		Model(artist).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create artist: %w", err)
	}

	return nil
}

// Update обновляет артиста
func (r *ArtistRepository) Update(artist *model.Artist) error {
	ctx := context.Background()

	// Сохраняем имя артиста точно как пришло с сайта
	// Никакой нормализации - только убираем лишние пробелы по краям
	artist.Name = strings.TrimSpace(artist.Name)

	_, err := r.db.NewUpdate().
		Model(artist).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update artist: %w", err)
	}

	return nil
}

// Delete удаляет артиста
func (r *ArtistRepository) Delete(id int) error {
	ctx := context.Background()

	_, err := r.db.NewDelete().
		Model((*model.Artist)(nil)).
		Where("artist_id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete artist: %w", err)
	}

	return nil
}

// GetActive возвращает активных артистов
func (r *ArtistRepository) GetActive() ([]model.Artist, error) {
	ctx := context.Background()
	var artists []model.Artist

	err := r.db.NewSelect().
		Model(&artists).
		Where("is_active = ?", true).
		Order("name ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query active artists: %w", err)
	}

	return artists, nil
}

// GetByGenderAndActive возвращает артистов по полу и активности
func (r *ArtistRepository) GetByGenderAndActive(gender model.Gender, active bool) ([]model.Artist, error) {
	ctx := context.Background()
	var artists []model.Artist

	err := r.db.NewSelect().
		Model(&artists).
		Where("gender = ? AND is_active = ?", gender, active).
		Order("name ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query artists by gender and active: %w", err)
	}

	return artists, nil
}
