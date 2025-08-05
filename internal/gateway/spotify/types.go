// Package spotify содержит типы для работы с Spotify API.
package spotify

// Track представляет трек из плейлиста Spotify
type Track struct {
	ID     string // Spotify Track ID
	Title  string // Название трека
	Artist string // Исполнитель
}

// PlaylistInfo содержит информацию о плейлисте Spotify
type PlaylistInfo struct {
	ID          string
	Name        string
	Description string
	TotalTracks int
	Public      bool
	Owner       string
}
