// Package playlist содержит типы данных для работы с плейлистами.
package playlist

import "gemfactory/internal/gateway/spotify"

// PlaylistService определяет интерфейс для работы с плейлистами
type PlaylistService interface {
	// GetRandomTrack возвращает случайный трек из плейлиста
	GetRandomTrack() (*spotify.Track, error)

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
	GetRandomTrack() (*spotify.Track, error)

	// GetTotalTracks возвращает общее количество треков в плейлиста
	GetTotalTracks() int

	// GetPlaylistInfo возвращает информацию о плейлисте
	GetPlaylistInfo() (*spotify.PlaylistInfo, error)

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

// BotAPIInterface определяет интерфейс для отправки сообщений через бота
type BotAPIInterface interface {
	// SendMessageToAdmin отправляет сообщение администратору
	SendMessageToAdmin(message string) error
}
