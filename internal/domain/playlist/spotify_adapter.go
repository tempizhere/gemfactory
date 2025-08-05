// Package playlist содержит адаптеры для работы с Spotify API.
package playlist

import (
	"gemfactory/internal/gateway/spotify"
)

// SpotifyAdapter адаптирует Spotify клиент из gateway слоя к domain интерфейсу
type SpotifyAdapter struct {
	client interface {
		GetPlaylistTracks(playlistURL string) ([]*spotify.Track, error)
		GetPlaylistInfo(playlistURL string) (*spotify.PlaylistInfo, error)
	}
}

// NewSpotifyAdapter создает новый адаптер для Spotify клиента
func NewSpotifyAdapter(client interface {
	GetPlaylistTracks(playlistURL string) ([]*spotify.Track, error)
	GetPlaylistInfo(playlistURL string) (*spotify.PlaylistInfo, error)
}) SpotifyClientInterface {
	return &SpotifyAdapter{client: client}
}

// GetPlaylistTracks получает треки из публичного плейлиста
func (a *SpotifyAdapter) GetPlaylistTracks(playlistURL string) ([]*spotify.Track, error) {
	gatewayTracks, err := a.client.GetPlaylistTracks(playlistURL)
	if err != nil {
		return nil, err
	}

	// Возвращаем треки напрямую, так как теперь используем общие типы
	return gatewayTracks, nil
}

// GetPlaylistInfo получает информацию о плейлисте
func (a *SpotifyAdapter) GetPlaylistInfo(playlistURL string) (*spotify.PlaylistInfo, error) {
	gatewayPlaylistInfo, err := a.client.GetPlaylistInfo(playlistURL)
	if err != nil {
		return nil, err
	}

	// Возвращаем информацию о плейлисте напрямую, так как теперь используем общие типы
	return gatewayPlaylistInfo, nil
}
