// Package spotify реализует интерфейсы для работы с Spotify Web API.
package spotify

import "gemfactory/internal/types"

// Interface определяет интерфейс для работы с Spotify API
type Interface interface {
	// ExtractPlaylistID извлекает ID плейлиста из URL
	ExtractPlaylistID(playlistURL string) (string, error)

	// GetPlaylistTracks получает треки из публичного плейлиста
	GetPlaylistTracks(playlistURL string) ([]*types.SpotifyTrack, error)

	// GetPlaylistInfo получает информацию о плейлисте
	GetPlaylistInfo(playlistURL string) (*types.SpotifyPlaylistInfo, error)
}
