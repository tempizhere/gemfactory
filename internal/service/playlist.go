// Package service содержит бизнес-логику приложения.
package service

import (
	"fmt"
	"gemfactory/internal/external/spotify"
	"gemfactory/internal/model"
	"gemfactory/internal/storage/repository"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// PlaylistService содержит бизнес-логику для работы с плейлистами
type PlaylistService struct {
	playlistRepo  model.PlaylistTracksRepository
	configRepo    model.ConfigRepository
	spotifyClient spotify.Client
	logger        *zap.Logger
}

// NewPlaylistService создает новый сервис плейлистов
func NewPlaylistService(db *bun.DB, spotifyClient spotify.Client, logger *zap.Logger) *PlaylistService {
	return &PlaylistService{
		playlistRepo:  repository.NewPlaylistTracksRepository(db, logger),
		configRepo:    repository.NewConfigRepository(db, logger),
		spotifyClient: spotifyClient,
		logger:        logger,
	}
}

// ReloadPlaylist перезагружает плейлист из Spotify
func (s *PlaylistService) ReloadPlaylist() error {
	// Получаем URL плейлиста из конфигурации
	playlistURL, err := s.configRepo.Get("PLAYLIST_URL")
	if err != nil {
		return fmt.Errorf("failed to get playlist URL from config: %w", err)
	}

	if playlistURL == nil || playlistURL.Value == "" {
		return fmt.Errorf("playlist URL not configured")
	}

	// Извлекаем Spotify ID из URL
	spotifyID := s.extractSpotifyID(playlistURL.Value)
	s.logger.Info("Extracted Spotify ID in PlaylistService", zap.String("playlist_url", playlistURL.Value), zap.String("spotify_id", spotifyID))
	if spotifyID == "" {
		return fmt.Errorf("failed to extract Spotify ID from playlist URL")
	}

	s.logger.Info("Starting playlist reload", zap.String("spotify_id", spotifyID))

	// Получаем информацию о плейлисте
	playlistInfo, err := s.spotifyClient.GetPlaylistInfo(playlistURL.Value)
	if err != nil {
		return fmt.Errorf("failed to get playlist info: %w", err)
	}

	s.logger.Info("Got playlist info",
		zap.String("name", playlistInfo.Name),
		zap.Int("track_count", playlistInfo.TrackCount))

	// Очищаем старые треки
	err = s.playlistRepo.DeleteBySpotifyID(spotifyID)
	if err != nil {
		return fmt.Errorf("failed to delete old tracks: %w", err)
	}

	s.logger.Info("Deleted old tracks")

	// Получаем треки из плейлиста
	tracks, err := s.spotifyClient.GetPlaylistTracks(playlistURL.Value)
	if err != nil {
		return fmt.Errorf("failed to get playlist tracks: %w", err)
	}

	s.logger.Info("Got tracks from Spotify", zap.Int("count", len(tracks)))

	// Сохраняем треки в базу данных
	savedCount := 0
	for _, track := range tracks {
		playlistTrack := &model.PlaylistTracks{
			SpotifyID:  spotifyID,
			TrackID:    track.ID,
			Artist:     track.Artist,
			Title:      track.Title,
			Album:      "", // Пока не получаем из Spotify API
			DurationMs: 0,  // Пока не получаем из Spotify API
			AddedAt:    time.Now(),
		}

		err = s.playlistRepo.Create(playlistTrack)
		if err != nil {
			s.logger.Error("Failed to save track",
				zap.String("track_id", track.ID),
				zap.String("artist", track.Artist),
				zap.String("title", track.Title),
				zap.Error(err))
			continue
		}

		savedCount++
	}

	s.logger.Info("Playlist reload completed",
		zap.String("spotify_id", spotifyID),
		zap.Int("saved_tracks", savedCount),
		zap.Int("total_tracks", len(tracks)))

	return nil
}

// GetPlaylistInfo возвращает информацию о плейлисте
func (s *PlaylistService) GetPlaylistInfo() (*spotify.PlaylistInfo, error) {
	// Получаем URL плейлиста из конфигурации
	playlistURL, err := s.configRepo.Get("PLAYLIST_URL")
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist URL from config: %w", err)
	}

	if playlistURL == nil || playlistURL.Value == "" {
		return nil, fmt.Errorf("playlist URL not configured")
	}

	// Получаем информацию о плейлисте
	playlistInfo, err := s.spotifyClient.GetPlaylistInfo(playlistURL.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist info: %w", err)
	}

	return playlistInfo, nil
}

// extractSpotifyID извлекает Spotify ID из URL плейлиста
func (s *PlaylistService) extractSpotifyID(playlistURL string) string {
	// Поддерживаем разные форматы URL:
	// https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M
	// spotify:playlist:37i9dQZF1DXcBWIGoYBM5M

	if strings.HasPrefix(playlistURL, "spotify:playlist:") {
		return strings.TrimPrefix(playlistURL, "spotify:playlist:")
	}

	if strings.Contains(playlistURL, "open.spotify.com/playlist/") {
		parts := strings.Split(playlistURL, "/playlist/")
		if len(parts) != 2 {
			return ""
		}
		// Убираем возможные параметры после ID
		playlistID := strings.Split(parts[1], "?")[0]
		return playlistID
	}

	return ""
}
