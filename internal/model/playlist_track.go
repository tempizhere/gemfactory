// Package model содержит модели данных.
//
// Группа: ENTITIES - Основные сущности
// Содержит: PlaylistTrack, PlaylistTrackRepository
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// PlaylistTracks представляет трек в плейлисте
type PlaylistTracks struct {
	bun.BaseModel `bun:"table:gemfactory.playlist_tracks"`

	ID        int       `bun:"id,pk,autoincrement" json:"id"`
	SpotifyID string    `bun:"spotify_id,notnull" json:"spotify_id"`
	TrackID   string    `bun:"track_id,notnull" json:"track_id"`
	Artist    string    `bun:"artist,notnull" json:"artist"`
	Title     string    `bun:"title,notnull" json:"title"`
	AddedAt   time.Time `bun:"added_at,notnull,default:current_timestamp" json:"added_at"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// PlaylistTracksRepository определяет интерфейс для работы с треками плейлиста
type PlaylistTracksRepository interface {
	GetBySpotifyID(spotifyID string) ([]PlaylistTracks, error)
	GetRandomTrack(spotifyID string, excludeTrackIDs []string) (*PlaylistTracks, error)
	Create(track *PlaylistTracks) error
	Update(track *PlaylistTracks) error
	Delete(id int) error
	DeleteBySpotifyID(spotifyID string) error
	GetAllBySpotifyID(spotifyID string) ([]PlaylistTracks, error)
}
