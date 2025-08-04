// Package playlist содержит типы данных для работы с плейлистами.
package playlist

import "gemfactory/internal/types"

// PlaylistService определяет интерфейс для работы с плейлистами
type PlaylistService interface {
	// GetRandomTrack возвращает случайный трек из плейлиста
	GetRandomTrack() (*types.SpotifyTrack, error)

	// GetTotalTracks возвращает общее количество треков в плейлисте
	GetTotalTracks() int

	// LoadPlaylistFromSpotify загружает плейлист из Spotify по URL
	LoadPlaylistFromSpotify(playlistURL string) error

	// IsLoaded проверяет, загружен ли плейлист
	IsLoaded() bool
}

// PlaylistManager определяет интерфейс для управления плейлистами
type PlaylistManager interface {
	// GetRandomTrack возвращает случайный трек из плейлиста
	GetRandomTrack() (*types.SpotifyTrack, error)

	// GetTotalTracks возвращает общее количество треков в плейлисте
	GetTotalTracks() int

	// GetPlaylistInfo возвращает информацию о плейлисте
	GetPlaylistInfo() (*types.SpotifyPlaylistInfo, error)

	// LoadPlaylistFromSpotify загружает плейлист из Spotify по URL
	LoadPlaylistFromSpotify(playlistURL string) error

	// LoadPlaylistFromStorage загружает плейлист из постоянного хранилища
	LoadPlaylistFromStorage() error

	// SavePlaylistToStorage сохраняет плейлист в постоянное хранилище
	SavePlaylistToStorage() error

	// IsLoaded проверяет, загружен ли плейлист
	IsLoaded() bool

	// Clear очищает плейлист
	Clear()
}
