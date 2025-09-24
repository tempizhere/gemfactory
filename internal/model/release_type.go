// Package model содержит модели данных.
//
// Группа: ENTITIES - Основные сущности
// Содержит: ReleaseType, ReleaseTypeRepository
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// ReleaseTypeModel представляет тип релиза
type ReleaseTypeModel struct {
	bun.BaseModel `bun:"table:release_types"`

	ReleaseTypeID int       `bun:"release_type_id,pk,autoincrement" json:"release_type_id"`
	Name          string    `bun:"name,unique,notnull" json:"name"`
	Description   string    `bun:"description" json:"description"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// Validate проверяет валидность типа релиза
func (rt *ReleaseTypeModel) Validate() error {
	var errors ValidationErrors

	if err := ValidateRequired("name", rt.Name); err != nil {
		errors = append(errors, err.(ValidationError))
	}

	if err := ValidateLength("name", rt.Name, 1, 20); err != nil {
		errors = append(errors, err.(ValidationError))
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ReleaseTypeRepository определяет интерфейс для работы с типами релизов
type ReleaseTypeRepository interface {
	Repository[ReleaseTypeModel]
	GetByName(name string) (*ReleaseTypeModel, error)
	GetAll() ([]ReleaseTypeModel, error)
}
