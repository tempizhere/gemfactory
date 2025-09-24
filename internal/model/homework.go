// Package model содержит модели данных.
//
// Группа: ENTITIES - Основные сущности
// Содержит: Homework, HomeworkRepository
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Homework представляет домашнее задание
type Homework struct {
	bun.BaseModel `bun:"table:gemfactory.homeworks"`

	HomeworkID int       `bun:"homework_id,pk,autoincrement" json:"homework_id"`
	UserID     int64     `bun:"user_id,notnull" json:"user_id"`
	TrackID    string    `bun:"track_id,notnull" json:"track_id"`
	Artist     string    `bun:"artist,notnull" json:"artist"`
	Title      string    `bun:"title,notnull" json:"title"`
	PlayCount  int       `bun:"play_count,notnull,default:1" json:"play_count"`
	Completed  bool      `bun:"completed,notnull,default:false" json:"completed"`
	CreatedAt  time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt  time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// HomeworkRepository определяет интерфейс для работы с домашними заданиями
type HomeworkRepository interface {
	GetByUserID(userID int64) ([]Homework, error)
	GetActiveByUserID(userID int64) (*Homework, error)
	Create(homework *Homework) error
	Update(homework *Homework) error
	Delete(id int) error
	MarkCompleted(id int) error
	GetRandomTrack() (*Homework, error)
	CanRequestHomework(userID int64) (bool, error)
	GetLastRequestTime(userID int64) (*time.Time, error)
}
