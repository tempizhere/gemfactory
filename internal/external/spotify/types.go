// Package spotify содержит типы для работы с Spotify API.
package spotify

import "time"

// Track представляет трек из плейлиста Spotify
type Track struct {
	ID     string // Spotify Track ID
	Title  string // Название трека
	Artist string // Исполнитель
}

// TrackInfo представляет информацию о треке
type TrackInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Artists  []string `json:"artists"`
	Duration int      `json:"duration"`
}

// PlaylistInfo представляет информацию о плейлисте
type PlaylistInfo struct {
	SpotifyID   string    `json:"spotify_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Owner       string    `json:"owner"`
	TrackCount  int       `json:"track_count"`
	LastUpdated time.Time `json:"last_updated"`
}
