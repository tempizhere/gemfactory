// Package model содержит утилиты для моделей.
//
// Группа: UTILS - Общие утилиты
// Содержит: CachedUtils, утилиты для работы со строками
package model

import (
	"html"
	"regexp"
	"strings"
	"sync"
)

// CachedUtils содержит кэшированные утилиты
type CachedUtils struct {
	// Кэшированные регулярные выражения
	whitespaceRegex *regexp.Regexp
	cleanupRegex    *regexp.Regexp

	// Кэш для очищенных строк
	cleanedCache sync.Map
}

var (
	// Глобальный экземпляр утилит
	utils *CachedUtils
	once  sync.Once
)

// GetUtils возвращает глобальный экземпляр утилит
func GetUtils() *CachedUtils {
	once.Do(func() {
		utils = &CachedUtils{
			whitespaceRegex: regexp.MustCompile(`\s+`),
			cleanupRegex:    regexp.MustCompile(`[\[\](){}]`),
		}
	})
	return utils
}

// CleanText очищает текст от лишних символов и пробелов
func (u *CachedUtils) CleanText(text string) string {
	if text == "" {
		return ""
	}

	// Проверяем кэш
	if cached, ok := u.cleanedCache.Load(text); ok {
		return cached.(string)
	}

	// Очищаем текст
	cleaned := strings.TrimSpace(text)
	cleaned = u.cleanupRegex.ReplaceAllString(cleaned, "")
	cleaned = u.whitespaceRegex.ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	// Сохраняем в кэш
	u.cleanedCache.Store(text, cleaned)

	return cleaned
}

// EscapeHTML экранирует HTML символы
func (u *CachedUtils) EscapeHTML(text string) string {
	return html.EscapeString(text)
}

// NormalizeString нормализует строку (приводит к нижнему регистру и очищает)
func (u *CachedUtils) NormalizeString(text string) string {
	return strings.ToLower(u.CleanText(text))
}

// TruncateString обрезает строку до указанной длины
func (u *CachedUtils) TruncateString(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

// Contains проверяет, содержит ли слайс элемент
func (u *CachedUtils) Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates удаляет дубликаты из слайса строк
func (u *CachedUtils) RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// SplitAndClean разделяет строку по разделителю и очищает каждый элемент
func (u *CachedUtils) SplitAndClean(text, separator string) []string {
	if text == "" {
		return nil
	}

	parts := strings.Split(text, separator)
	var result []string

	for _, part := range parts {
		cleaned := u.CleanText(part)
		if cleaned != "" {
			result = append(result, cleaned)
		}
	}

	return result
}

// JoinNonEmpty объединяет непустые строки
func (u *CachedUtils) JoinNonEmpty(separator string, parts ...string) string {
	var nonEmpty []string

	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}

	return strings.Join(nonEmpty, separator)
}

// ClearCache очищает кэш утилит
func (u *CachedUtils) ClearCache() {
	u.cleanedCache.Range(func(key, value interface{}) bool {
		u.cleanedCache.Delete(key)
		return true
	})
}
