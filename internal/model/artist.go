// Package model содержит модели данных.
//
// Группа: ENTITIES - Основные сущности
// Содержит: Artist, ArtistRepository
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Artist представляет артиста
type Artist struct {
	bun.BaseModel `bun:"table:artists"`

	ArtistID  int       `bun:"artist_id,pk,autoincrement" json:"artist_id"`
	Name      string    `bun:"name,unique,notnull" json:"name"`
	Gender    Gender    `bun:"gender,notnull,default:'male'" json:"gender"`
	IsActive  bool      `bun:"is_active,notnull,default:true" json:"is_active"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// Validate проверяет валидность артиста
func (a *Artist) Validate() error {
	var errors ValidationErrors

	if err := ValidateRequired("name", a.Name); err != nil {
		errors = append(errors, err.(ValidationError))
	}

	if err := ValidateLength("name", a.Name, 1, 100); err != nil {
		errors = append(errors, err.(ValidationError))
	}

	if !a.Gender.IsValid() {
		errors = append(errors, ValidationError{Field: "gender", Message: "invalid gender value"})
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// IsFemale проверяет, является ли артист женским
func (a *Artist) IsFemale() bool {
	return a.Gender == GenderFemale
}

// SetGender устанавливает пол артиста
func (a *Artist) SetGender(isFemale bool) {
	if isFemale {
		a.Gender = GenderFemale
	} else {
		a.Gender = GenderMale
	}
}

// GetDisplayName возвращает отображаемое имя артиста
func (a *Artist) GetDisplayName() string {
	return GetUtils().CleanText(a.Name)
}

// ArtistRepository определяет интерфейс для работы с артистами
type ArtistRepository interface {
	Repository[Artist]
	GetByGender(gender Gender) ([]Artist, error)
	GetByName(name string) (*Artist, error)
	GetActive() ([]Artist, error)
	GetByGenderAndActive(gender Gender, active bool) ([]Artist, error)
}
