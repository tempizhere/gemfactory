// Package playlist содержит менеджер для работы с плейлистами.
package playlist

import (
	"fmt"
	"math/rand"
	"sync"

	"gemfactory/internal/gateway/spotify"

	"go.uber.org/zap"
)

// Manager управляет плейлистами
type Manager struct {
	tracks        []*spotify.Track
	playlistInfo  *spotify.PlaylistInfo
	playlistURL   string
	mu            sync.RWMutex
	loaded        bool
	logger        *zap.Logger
	storageDir    string
	spotifyClient SpotifyClientInterface
}

var _ PlaylistManager = (*Manager)(nil)

// NewManager создает новый менеджер плейлистов
func NewManager(logger *zap.Logger, storageDir string, spotifyClient SpotifyClientInterface) *Manager {
	return &Manager{
		tracks:        make([]*spotify.Track, 0),
		logger:        logger,
		storageDir:    storageDir,
		spotifyClient: spotifyClient,
	}
}

// LoadPlaylistFromSpotify загружает плейлист из Spotify по URL
func (m *Manager) LoadPlaylistFromSpotify(playlistURL string) error {
	if m.spotifyClient == nil {
		return fmt.Errorf("spotify client is not available")
	}

	m.logger.Info("Loading playlist from Spotify", zap.String("playlist_url", playlistURL))

	// Загружаем треки плейлиста
	tracks, err := m.spotifyClient.GetPlaylistTracks(playlistURL)
	if err != nil {
		m.logger.Error("Failed to get playlist tracks from Spotify",
			zap.String("playlist_url", playlistURL), zap.Error(err))
		return fmt.Errorf("failed to get playlist tracks from Spotify: %w", err)
	}

	// Загружаем информацию о плейлисте
	playlistInfo, err := m.spotifyClient.GetPlaylistInfo(playlistURL)
	if err != nil {
		m.logger.Error("Failed to get playlist info from Spotify",
			zap.String("playlist_url", playlistURL), zap.Error(err))
		return fmt.Errorf("failed to get playlist info from Spotify: %w", err)
	}

	// Подготавливаем новые треки
	var newTracks []*spotify.Track
	for _, track := range tracks {
		if track.Title != "" && track.Artist != "" {
			newTracks = append(newTracks, track)
		}
	}

	// Только после успешной подготовки всех данных обновляем состояние
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tracks = newTracks
	m.playlistInfo = playlistInfo
	m.playlistURL = playlistURL
	m.loaded = true

	m.logger.Info("Playlist loaded from Spotify successfully",
		zap.String("playlist_url", playlistURL), zap.Int("tracks_count", len(m.tracks)))

	return nil
}

// GetRandomTrack возвращает случайный трек из плейлиста
func (m *Manager) GetRandomTrack() (*spotify.Track, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return nil, fmt.Errorf("playlist not loaded")
	}

	if len(m.tracks) == 0 {
		return nil, fmt.Errorf("playlist is empty")
	}

	// Генерируем случайный индекс
	index := m.randomInt(len(m.tracks))

	track := m.tracks[index]
	m.logger.Debug("Selected random track",
		zap.String("artist", track.Artist),
		zap.String("title", track.Title),
		zap.Int("index", index))

	return track, nil
}

// GetTotalTracks возвращает общее количество треков в плейлисте
func (m *Manager) GetTotalTracks() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tracks)
}

// GetPlaylistInfo возвращает информацию о плейлисте
func (m *Manager) GetPlaylistInfo() (*spotify.PlaylistInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return nil, fmt.Errorf("playlist not loaded")
	}

	if m.playlistInfo == nil {
		return nil, fmt.Errorf("playlist info not available")
	}

	return m.playlistInfo, nil
}

// IsLoaded проверяет, загружен ли плейлист
func (m *Manager) IsLoaded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loaded
}

// Clear очищает плейлист
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tracks = make([]*spotify.Track, 0)
	m.playlistInfo = nil
	m.playlistURL = ""
	m.loaded = false
	m.logger.Info("Playlist cleared")
}

// LoadPlaylistFromStorage загружает плейлист из постоянного хранилища
func (m *Manager) LoadPlaylistFromStorage() error {
	// Плейлисты теперь загружаются только из Spotify
	m.logger.Info("Playlist storage loading is not supported, use LoadPlaylistFromSpotify instead")
	return nil
}

// SavePlaylistToStorage сохраняет плейлист в постоянное хранилище
func (m *Manager) SavePlaylistToStorage() error {
	// Плейлисты теперь сохраняются только в памяти
	m.logger.Info("Playlist storage saving is not supported")
	return nil
}

// randomInt генерирует случайное число
func (m *Manager) randomInt(n int) int {
	return rand.Intn(n)
}
