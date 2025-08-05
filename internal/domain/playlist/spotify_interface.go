// Package playlist содержит интерфейсы для работы с Spotify API.
package playlist

import "gemfactory/internal/gateway/spotify"

// SpotifyClientInterface определяет интерфейс для работы с Spotify API
type SpotifyClientInterface interface {
	// GetPlaylistTracks получает треки из публичного плейлиста
	GetPlaylistTracks(playlistURL string) ([]*spotify.Track, error)
	// GetPlaylistInfo получает информацию о плейлисте
	GetPlaylistInfo(playlistURL string) (*spotify.PlaylistInfo, error)
}
