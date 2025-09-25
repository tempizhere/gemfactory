// Package repository содержит репозитории для работы с базой данных.
package repository

import (
	"context"
	"fmt"

	"gemfactory/internal/model"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// PlaylistTracksRepository реализует интерфейс для работы с треками плейлиста
type PlaylistTracksRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewPlaylistTracksRepository создает новый репозиторий для треков плейлиста
func NewPlaylistTracksRepository(db *bun.DB, logger *zap.Logger) *PlaylistTracksRepository {
	return &PlaylistTracksRepository{
		db:     db,
		logger: logger,
	}
}

// GetBySpotifyID возвращает все треки плейлиста по Spotify ID
func (r *PlaylistTracksRepository) GetBySpotifyID(spotifyID string) ([]model.PlaylistTracks, error) {
	ctx := context.Background()
	var tracks []model.PlaylistTracks

	err := r.db.NewSelect().
		Model(&tracks).
		Where("spotify_id = ?", spotifyID).
		Order("added_at ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get playlist tracks: %w", err)
	}

	return tracks, nil
}

// GetRandomTrack возвращает случайный трек из плейлиста, исключая уже выданные
func (r *PlaylistTracksRepository) GetRandomTrack(spotifyID string, excludeTrackIDs []string) (*model.PlaylistTracks, error) {
	ctx := context.Background()
	track := new(model.PlaylistTracks)

	r.logger.Info("GetRandomTrack called", zap.String("spotify_id", spotifyID), zap.Strings("exclude_track_ids", excludeTrackIDs))

	query := r.db.NewSelect().
		Model(track).
		Where("spotify_id = ?", spotifyID)

	// Исключаем уже выданные треки
	if len(excludeTrackIDs) > 0 {
		query = query.Where("track_id NOT IN (?)", bun.In(excludeTrackIDs))
	}

	err := query.
		OrderExpr("RANDOM()").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			r.logger.Info("No tracks found in playlist", zap.String("spotify_id", spotifyID))
			return nil, nil
		}
		r.logger.Error("Failed to get random track", zap.Error(err))
		return nil, fmt.Errorf("failed to get random track: %w", err)
	}

	r.logger.Info("Found random track", zap.String("track_id", track.TrackID), zap.String("artist", track.Artist), zap.String("title", track.Title))
	return track, nil
}

// Create создает новый трек в плейлисте
func (r *PlaylistTracksRepository) Create(track *model.PlaylistTracks) error {
	ctx := context.Background()

	_, err := r.db.NewInsert().
		Model(track).
		On("CONFLICT (spotify_id, track_id) DO UPDATE").
		Set("artist = EXCLUDED.artist").
		Set("title = EXCLUDED.title").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create playlist track: %w", err)
	}

	return nil
}

// Update обновляет трек в плейлисте
func (r *PlaylistTracksRepository) Update(track *model.PlaylistTracks) error {
	ctx := context.Background()

	_, err := r.db.NewUpdate().
		Model(track).
		Where("id = ?", track.ID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update playlist track: %w", err)
	}

	return nil
}

// Delete удаляет трек из плейлиста
func (r *PlaylistTracksRepository) Delete(id int) error {
	ctx := context.Background()

	_, err := r.db.NewDelete().
		Model((*model.PlaylistTracks)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete playlist track: %w", err)
	}

	return nil
}

// DeleteBySpotifyID удаляет все треки плейлиста по Spotify ID
func (r *PlaylistTracksRepository) DeleteBySpotifyID(spotifyID string) error {
	ctx := context.Background()

	_, err := r.db.NewDelete().
		Model((*model.PlaylistTracks)(nil)).
		Where("spotify_id = ?", spotifyID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete playlist tracks by spotify_id: %w", err)
	}

	return nil
}

// GetAllBySpotifyID возвращает все треки плейлиста по Spotify ID
func (r *PlaylistTracksRepository) GetAllBySpotifyID(spotifyID string) ([]model.PlaylistTracks, error) {
	return r.GetBySpotifyID(spotifyID)
}
