// Package model содержит вспомогательные функции для работы с моделями.
//
// Группа: UTILS - Утилиты для моделей
// Содержит: CachedUtils, GetUtils
package model

import (
	"strings"
)

// CachedUtils предоставляет кэшированные утилиты для работы со строками
type CachedUtils struct {
	cache map[string]string
}

// CleanText очищает текст от лишних символов
func (u *CachedUtils) CleanText(text string) string {
	if cached, exists := u.cache[text]; exists {
		return cached
	}

	cleaned := strings.TrimSpace(text)
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")

	// Убираем множественные пробелы
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}

	u.cache[text] = cleaned
	return cleaned
}

// GetUtils возвращает глобальный экземпляр CachedUtils
func GetUtils() *CachedUtils {
	return &CachedUtils{
		cache: make(map[string]string),
	}
}
