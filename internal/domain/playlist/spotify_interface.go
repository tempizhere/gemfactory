// Package playlist содержит интерфейсы для работы с Spotify API.
package playlist

import "gemfactory/internal/types"

// SpotifyClientInterface определяет интерфейс для работы с Spotify API
type SpotifyClientInterface interface {
	// GetPlaylistTracks получает треки из публичного плейлиста
	GetPlaylistTracks(playlistURL string) ([]*types.SpotifyTrack, error)

	// GetPlaylistInfo получает информацию о плейлисте
	GetPlaylistInfo(playlistURL string) (*types.SpotifyPlaylistInfo, error)
}
