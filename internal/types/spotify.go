// Package types содержит общие типы для работы с внешними API.
package types

// SpotifyTrack представляет трек из плейлиста Spotify
type SpotifyTrack struct {
	ID     string // Spotify Track ID
	Title  string // Название трека
	Artist string // Исполнитель
}

// SpotifyPlaylistInfo содержит информацию о плейлисте Spotify
type SpotifyPlaylistInfo struct {
	ID          string
	Name        string
	Description string
	TotalTracks int
	Public      bool
	Owner       string
}
