// Package playlist содержит сервисы для работы с плейлистами.
package playlist

import (
	"fmt"
	"math/rand"
	"sync"

	"gemfactory/internal/gateway/spotify"

	"go.uber.org/zap"
)

// playlistServiceImpl реализует интерфейс PlaylistService
type playlistServiceImpl struct {
	tracks        []*spotify.Track
	mu            sync.RWMutex
	loaded        bool
	logger        *zap.Logger
	spotifyClient SpotifyClientInterface
}

// NewPlaylistService создает новый экземпляр PlaylistService
func NewPlaylistService(logger *zap.Logger, spotifyClient SpotifyClientInterface) PlaylistService {
	return &playlistServiceImpl{
		tracks:        make([]*spotify.Track, 0),
		logger:        logger,
		spotifyClient: spotifyClient,
	}
}

// LoadPlaylistFromSpotify загружает плейлист из Spotify по URL
func (p *playlistServiceImpl) LoadPlaylistFromSpotify(playlistURL string) error {
	if p.spotifyClient == nil {
		return fmt.Errorf("spotify client is not available")
	}

	p.logger.Info("Loading playlist from Spotify", zap.String("playlist_url", playlistURL))

	tracks, err := p.spotifyClient.GetPlaylistTracks(playlistURL)
	if err != nil {
		p.logger.Error("Failed to get playlist tracks from Spotify",
			zap.String("playlist_url", playlistURL), zap.Error(err))
		return fmt.Errorf("failed to get playlist tracks from Spotify: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Очищаем существующие треки
	p.tracks = make([]*spotify.Track, 0)

	// Добавляем новые треки
	for _, track := range tracks {
		if track.Title != "" && track.Artist != "" {
			p.tracks = append(p.tracks, track)
		}
	}

	p.loaded = true
	p.logger.Info("Playlist loaded from Spotify successfully",
		zap.String("playlist_url", playlistURL), zap.Int("tracks_count", len(p.tracks)))

	return nil
}

// GetRandomTrack возвращает случайный трек из плейлиста
func (p *playlistServiceImpl) GetRandomTrack() (*spotify.Track, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.loaded {
		return nil, fmt.Errorf("playlist not loaded")
	}

	if len(p.tracks) == 0 {
		return nil, fmt.Errorf("playlist is empty")
	}

	// Генерируем случайный индекс
	index := rand.Intn(len(p.tracks))

	track := p.tracks[index]
	p.logger.Debug("Selected random track",
		zap.String("artist", track.Artist),
		zap.String("title", track.Title),
		zap.Int("index", index))

	return track, nil
}

// GetTotalTracks возвращает общее количество треков в плейлисте
func (p *playlistServiceImpl) GetTotalTracks() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.tracks)
}

// IsLoaded проверяет, загружен ли плейлист
func (p *playlistServiceImpl) IsLoaded() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.loaded
}
