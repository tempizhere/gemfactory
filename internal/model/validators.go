// Package model содержит валидаторы для моделей.
//
// Группа: BASE - Базовые компоненты
// Содержит: Validator, ValidationError, ValidationErrors, валидаторы
package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Validator представляет интерфейс валидатора
type Validator interface {
	Validate() error
}

// ValidationError представляет ошибку валидации
type ValidationError struct {
	Field   string
	Message string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", ve.Field, ve.Message)
}

// ValidationErrors представляет множество ошибок валидации
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}

	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// HasErrors проверяет, есть ли ошибки валидации
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Common validators
var (
	// Regex для проверки email (если понадобится)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// Regex для проверки URL
	urlRegex = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)

)

// ValidateRequired проверяет, что поле не пустое
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return ValidationError{Field: field, Message: "is required"}
	}
	return nil
}

// ValidateLength проверяет длину строки
func ValidateLength(field, value string, min, max int) error {
	length := len(strings.TrimSpace(value))
	if length < min {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be at least %d characters", min)}
	}
	if length > max {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be at most %d characters", max)}
	}
	return nil
}

// ValidateEmail проверяет формат email
func ValidateEmail(field, email string) error {
	if email == "" {
		return nil // email не обязателен
	}
	if !emailRegex.MatchString(email) {
		return ValidationError{Field: field, Message: "invalid email format"}
	}
	return nil
}

// ValidateURL проверяет формат URL
func ValidateURL(field, url string) error {
	if url == "" {
		return nil // URL не обязателен
	}
	if !urlRegex.MatchString(url) {
		return ValidationError{Field: field, Message: "invalid URL format"}
	}
	return nil
}

// ValidateDate проверяет, что дата не в будущем
func ValidateDate(field string, date time.Time) error {
	if date.IsZero() {
		return ValidationError{Field: field, Message: "date is required"}
	}
	if date.After(time.Now()) {
		return ValidationError{Field: field, Message: "date cannot be in the future"}
	}
	return nil
}

// ValidatePositiveInt проверяет, что число положительное
func ValidatePositiveInt(field string, value int) error {
	if value <= 0 {
		return ValidationError{Field: field, Message: "must be positive"}
	}
	return nil
}

// ValidateNonNegativeInt проверяет, что число неотрицательное
func ValidateNonNegativeInt(field string, value int) error {
	if value < 0 {
		return ValidationError{Field: field, Message: "must be non-negative"}
	}
	return nil
}

// ValidateEnum проверяет, что значение входит в список допустимых
func ValidateEnum(field, value string, allowedValues []string) error {
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}
	return ValidationError{Field: field, Message: fmt.Sprintf("must be one of: %s", strings.Join(allowedValues, ", "))}
}
