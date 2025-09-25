// Package model содержит валидаторы для моделей данных.
//
// Группа: UTILS - Утилиты для валидации
// Содержит: ValidationError, ValidationErrors, ValidateRequired, ValidateLength, ValidateURL
package model

import (
	"regexp"
	"strings"
)

// ValidationError представляет ошибку валидации
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationErrors представляет коллекцию ошибок валидации
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// HasErrors проверяет, есть ли ошибки валидации
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

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
		return ValidationError{Field: field, Message: "is too short"}
	}
	if length > max {
		return ValidationError{Field: field, Message: "is too long"}
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

// Регулярные выражения для валидации
var (
	urlRegex = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
)
