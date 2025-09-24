// Package model содержит модели данных.
//
// Группа: ENTITIES - Основные сущности
// Содержит: HomeworkTracking, HomeworkTrackingRepository
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// HomeworkTracking представляет отслеживание выданных домашних заданий
type HomeworkTracking struct {
	bun.BaseModel `bun:"table:homeworks"`

	ID          int        `bun:"id,pk,autoincrement" json:"id"`
	UserID      int64      `bun:"user_id,notnull" json:"user_id"`
	TrackID     string     `bun:"track_id,notnull" json:"track_id"`
	SpotifyID   string     `bun:"spotify_id,notnull" json:"spotify_id"`
	PlayCount   int        `bun:"play_count,notnull,default:1" json:"play_count"`
	IssuedAt    time.Time  `bun:"issued_at,notnull,default:current_timestamp" json:"issued_at"`
	CompletedAt *time.Time `bun:"completed_at" json:"completed_at"`
	IsCompleted bool       `bun:"is_completed,notnull,default:false" json:"is_completed"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// HomeworkTrackingRepository определяет интерфейс для работы с отслеживанием домашних заданий
type HomeworkTrackingRepository interface {
	GetByUserID(userID int64) ([]HomeworkTracking, error)
	GetCompletedByUserID(userID int64) ([]HomeworkTracking, error)
	GetPendingByUserID(userID int64) ([]HomeworkTracking, error)
	GetAllPending() ([]HomeworkTracking, error)
	Create(tracking *HomeworkTracking) error
	Update(tracking *HomeworkTracking) error
	MarkCompleted(userID int64, trackID string, spotifyID string) error
	GetIssuedTrackIDs(userID int64, spotifyID string) ([]string, error)
	CanRequestHomework(userID int64) (bool, error)
	GetLastRequestTime(userID int64) (*time.Time, error)
}
