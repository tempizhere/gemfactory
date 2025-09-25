// Package model содержит базовые модели и интерфейсы.
//
// Группа: BASE - Базовые компоненты
// Содержит: BaseModel, Repository[T], TimestampedModel
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// BaseModel представляет базовую модель с общими полями
type BaseModel struct {
	bun.BaseModel

	ID        int       `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// TimestampedModel представляет модель с временными метками
type TimestampedModel struct {
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// Repository представляет базовый интерфейс репозитория
type Repository[T any] interface {
	GetByID(id int) (*T, error)
	Create(entity *T) error
	Update(entity *T) error
	Delete(id int) error
	GetAll() ([]T, error)
}
