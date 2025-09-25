// Package model содержит модели данных.
//
// Группа: ENTITIES - Основные сущности
// Содержит: Config, ConfigRepository
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Config представляет конфигурацию приложения
type Config struct {
	bun.BaseModel `bun:"table:gemfactory.config"`

	ID          int       `bun:"id,pk,autoincrement" json:"id"`
	Key         string    `bun:"key,unique,notnull" json:"key"`
	Value       string    `bun:"value,notnull" json:"value"`
	Description string    `bun:"description" json:"description"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// ConfigRepository определяет интерфейс для работы с конфигурацией
type ConfigRepository interface {
	Get(key string) (*Config, error)
	GetAll() ([]Config, error)
	Set(key, value string) error
	Delete(key string) error
	Reset() error
	GetDefaultConfig() map[string]string
}
