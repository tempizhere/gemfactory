// Package model содержит конфигурацию для релизов.
//
// Группа: UTILS - Утилиты для релизов
// Содержит: ReleaseConfig, конфигурация для обработки релизов
package model

import (
	"strings"
	"time"
)

// ReleaseConfig содержит конфигурацию для обработки релизов
type ReleaseConfig struct {
	months          []string
	monthToNumber   map[string]string
	dateFormat      string
	dateParseFormat string
	timeFormat      string
	timeParseFormat string
	kstToMSKDiff    time.Duration
}

// NewReleaseConfig создает новую конфигурацию релизов
func NewReleaseConfig() *ReleaseConfig {
	return &ReleaseConfig{
		months: []string{
			"january", "february", "march", "april", "may", "june",
			"july", "august", "september", "october", "november", "december",
		},
		monthToNumber: map[string]string{
			"january":   "01",
			"february":  "02",
			"march":     "03",
			"april":     "04",
			"may":       "05",
			"june":      "06",
			"july":      "07",
			"august":    "08",
			"september": "09",
			"october":   "10",
			"november":  "11",
			"december":  "12",
		},
		dateFormat:      "02.01.06",
		dateParseFormat: "January 2, 2006",
		timeFormat:      "15:04",
		timeParseFormat: "3 PM",
		kstToMSKDiff:    -6 * time.Hour,
	}
}

// Months возвращает список названий месяцев
func (c *ReleaseConfig) Months() []string {
	result := make([]string, len(c.months))
	copy(result, c.months)
	return result
}

// MonthToNumber возвращает числовое представление месяца
func (c *ReleaseConfig) MonthToNumber(month string) (string, bool) {
	num, ok := c.monthToNumber[strings.ToLower(month)]
	return num, ok
}

// DateFormat возвращает формат вывода для дат
func (c *ReleaseConfig) DateFormat() string {
	return c.dateFormat
}

// DateParseFormat возвращает формат для парсинга дат
func (c *ReleaseConfig) DateParseFormat() string {
	return c.dateParseFormat
}

// TimeFormat возвращает формат вывода для времени
func (c *ReleaseConfig) TimeFormat() string {
	return c.timeFormat
}

// TimeParseFormat возвращает формат для парсинга времени
func (c *ReleaseConfig) TimeParseFormat() string {
	return c.timeParseFormat
}

// KSTToMSKDiff возвращает разность времени между KST и MSK
func (c *ReleaseConfig) KSTToMSKDiff() time.Duration {
	return c.kstToMSKDiff
}

// CurrentYear возвращает текущий год
func CurrentYear() string {
	return time.Now().Format("2006")
}

// GetMonthName возвращает название месяца по номеру
func (c *ReleaseConfig) GetMonthName(monthNumber string) (string, bool) {
	for name, num := range c.monthToNumber {
		if num == monthNumber {
			return name, true
		}
	}
	return "", false
}

// IsValidMonth проверяет, является ли строка валидным месяцем
func (c *ReleaseConfig) IsValidMonth(month string) bool {
	_, ok := c.monthToNumber[strings.ToLower(month)]
	return ok
}

// GetMonthNumber возвращает номер месяца по названию
func (c *ReleaseConfig) GetMonthNumber(month string) string {
	if num, ok := c.monthToNumber[strings.ToLower(month)]; ok {
		return num
	}
	return ""
}

// FormatReleaseDate форматирует дату релиза
func (c *ReleaseConfig) FormatReleaseDate(date time.Time) string {
	return date.Format(c.dateFormat)
}

// ParseReleaseDate парсит дату релиза
func (c *ReleaseConfig) ParseReleaseDate(dateStr string) (time.Time, error) {
	return time.Parse(c.dateParseFormat, dateStr)
}

// FormatReleaseTime форматирует время релиза
func (c *ReleaseConfig) FormatReleaseTime(time time.Time) string {
	return time.Format(c.timeFormat)
}

// ParseReleaseTime парсит время релиза
func (c *ReleaseConfig) ParseReleaseTime(timeStr string) (time.Time, error) {
	return time.Parse(c.timeParseFormat, timeStr)
}

// ConvertKSTToMSK конвертирует время из KST в MSK
func (c *ReleaseConfig) ConvertKSTToMSK(kstTime time.Time) time.Time {
	return kstTime.Add(c.kstToMSKDiff)
}
