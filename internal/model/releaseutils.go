// Package model содержит утилиты для работы с релизами.
//
// Группа: UTILS - Утилиты для релизов
// Содержит: ReleaseUtils, FormatDateWithYear, утилиты для обработки релизов
package model

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ReleaseUtils содержит утилиты для работы с релизами
type ReleaseUtils struct {
	config *ReleaseConfig
}

// escapeHTML экранирует специальные HTML символы
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// EscapeHTMLTags экранирует HTML-теги в тексте для Telegram
func (u *ReleaseUtils) EscapeHTMLTags(text string) string {
	if text == "" {
		return text
	}

	// Экранируем HTML-теги, заменяя < и > на безопасные символы
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")

	return text
}

// NewReleaseUtils создает новый экземпляр утилит для релизов
func NewReleaseUtils() *ReleaseUtils {
	return &ReleaseUtils{
		config: NewReleaseConfig(),
	}
}

// ParseReleaseDate парсит дату релиза из различных форматов
func (u *ReleaseUtils) ParseReleaseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	dateStr = strings.ReplaceAll(dateStr, ":", "")
	dateStr = strings.TrimSpace(dateStr)

	if strings.Contains(dateStr, " at ") {
		parts := strings.Split(dateStr, " at ")
		if len(parts) > 0 {
			dateStr = strings.TrimSpace(parts[0])
		}
	}

	return u.parseDate(dateStr)
}

// ParseReleaseDateWithYear парсит дату релиза с указанным годом
func (u *ReleaseUtils) ParseReleaseDateWithYear(dateStr string, year string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	dateStr = strings.ReplaceAll(dateStr, ":", "")
	dateStr = strings.TrimSpace(dateStr)

	if strings.Contains(dateStr, " at ") {
		parts := strings.Split(dateStr, " at ")
		if len(parts) > 0 {
			dateStr = strings.TrimSpace(parts[0])
		}
	}

	return u.parseDateWithYear(dateStr, year)
}

// parseDate парсит строку даты с годом или без
func (u *ReleaseUtils) parseDate(dateStr string) (time.Time, error) {
	if strings.Contains(dateStr, ".") && len(strings.Split(dateStr, ".")) == 3 {
		parts := strings.Split(dateStr, ".")
		if len(parts) == 3 {
			day := parts[0]
			month := parts[1]
			year := parts[2]

			// Преобразуем год из двухзначного в четырехзначный
			if len(year) == 2 {
				yearInt, err := strconv.Atoi(year)
				if err == nil {
					// Предполагаем, что годы 00-30 это 2000-2030, а 31-99 это 1931-1999
					if yearInt <= 30 {
						year = fmt.Sprintf("20%s", year)
					} else {
						year = fmt.Sprintf("19%s", year)
					}
				}
			}

			// Формируем дату в формате "2006-01-02"
			dateStr = fmt.Sprintf("%s-%s-%s", year, month, day)
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				return parsedDate, nil
			}
		}
	}

	isDate := false
	for _, month := range u.config.Months() {
		if strings.HasPrefix(strings.ToLower(dateStr), month) {
			isDate = true
			break
		}
	}
	if !isDate {
		return time.Time{}, fmt.Errorf("invalid date string: %s", dateStr)
	}

	if strings.Contains(dateStr, ",") {
		return u.parseDateWithComma(dateStr)
	}

	parts := strings.Fields(dateStr)
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
	}

	if len(parts) > 2 && strings.Contains(parts[len(parts)-1], "20") {
		dateStr = strings.Join(parts, " ")
		parsedDate, err := time.Parse("January 2 2006", dateStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
		}
		return parsedDate, nil
	}

	// Формат без года, добавляем текущий год
	dateStr = strings.Join(parts[:2], " ") + " " + CurrentYear()
	parsedDate, err := time.Parse("January 2 2006", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
	}
	return parsedDate, nil
}

// parseDateWithYear парсит строку даты с указанным годом
func (u *ReleaseUtils) parseDateWithYear(dateStr string, year string) (time.Time, error) {
	if strings.Contains(dateStr, ".") && len(strings.Split(dateStr, ".")) == 3 {
		parts := strings.Split(dateStr, ".")
		if len(parts) == 3 {
			day := parts[0]
			month := parts[1]
			yearFromDate := parts[2]

			// Если год в дате двухзначный, используем его, иначе используем переданный год
			if len(yearFromDate) == 2 {
				yearInt, err := strconv.Atoi(yearFromDate)
				if err == nil {
					// Предполагаем, что годы 00-30 это 2000-2030, а 31-99 это 1931-1999
					if yearInt <= 30 {
						year = fmt.Sprintf("20%s", yearFromDate)
					} else {
						year = fmt.Sprintf("19%s", yearFromDate)
					}
				}
			} else {
				year = yearFromDate
			}

			// Формируем дату в формате "2006-01-02"
			dateStr = fmt.Sprintf("%s-%s-%s", year, month, day)
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				return parsedDate, nil
			}
		}
	}

	isDate := false
	for _, month := range u.config.Months() {
		if strings.HasPrefix(strings.ToLower(dateStr), month) {
			isDate = true
			break
		}
	}
	if !isDate {
		return time.Time{}, fmt.Errorf("invalid date string: %s", dateStr)
	}

	if strings.Contains(dateStr, ",") {
		return u.parseDateWithComma(dateStr)
	}

	parts := strings.Fields(dateStr)
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
	}

	if len(parts) > 2 && strings.Contains(parts[len(parts)-1], "20") {
		dateStr = strings.Join(parts, " ")
		parsedDate, err := time.Parse("January 2 2006", dateStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
		}
		return parsedDate, nil
	}

	// Формат без года, добавляем указанный год
	dateStr = strings.Join(parts[:2], " ") + " " + year
	parsedDate, err := time.Parse("January 2 2006", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
	}
	return parsedDate, nil
}

// parseDateWithComma парсит строку даты с запятыми
func (u *ReleaseUtils) parseDateWithComma(dateStr string) (time.Time, error) {
	parsedDate, err := time.Parse(u.config.DateParseFormat(), dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
	}
	return parsedDate, nil
}

// FormatReleaseDate форматирует дату релиза
func (u *ReleaseUtils) FormatReleaseDate(date time.Time) string {
	return date.Format(u.config.DateFormat())
}

// ParseReleaseTime парсит время релиза из различных форматов
func (u *ReleaseUtils) ParseReleaseTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	timeStr = strings.ReplaceAll(timeStr, "KST", "")
	timeStr = strings.TrimSpace(timeStr)
	if strings.Contains(timeStr, "at") {
		parts := strings.Split(timeStr, "at")
		if len(parts) > 1 {
			timeStr = strings.TrimSpace(parts[1])
		} else {
			return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
		}
	}

	parsedTime, err := time.Parse(u.config.TimeParseFormat(), timeStr)
	if err != nil {
		parsedTime, err = time.Parse("3:04 PM", timeStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse time '%s': %w", timeStr, err)
		}
	}

	return parsedTime, nil
}

// FormatReleaseTime форматирует время релиза
func (u *ReleaseUtils) FormatReleaseTime(time time.Time) string {
	return time.Format(u.config.TimeFormat())
}

// ConvertKSTToMSK конвертирует время из KST в MSK
func (u *ReleaseUtils) ConvertKSTToMSK(kstTime time.Time) time.Time {
	return kstTime.Add(u.config.KSTToMSKDiff())
}

// ConvertKSTToMSKString конвертирует строку времени KST в MSK
func (u *ReleaseUtils) ConvertKSTToMSKString(kstTimeStr string) (string, error) {
	if kstTimeStr == "" {
		return "", fmt.Errorf("empty KST time")
	}

	parsedTime, err := time.Parse(u.config.TimeFormat(), kstTimeStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse KST time '%s': %w", kstTimeStr, err)
	}

	mskTime := parsedTime.Add(u.config.KSTToMSKDiff())
	return mskTime.Format(u.config.TimeFormat()), nil
}

// CleanLink очищает YouTube ссылку
func (u *ReleaseUtils) CleanLink(link string) string {
	if link == "" {
		return ""
	}

	// Удаляем ссылки на каналы
	if strings.HasPrefix(link, "https://www.youtube.com/@") || strings.HasPrefix(link, "https://youtube.com/@") ||
		strings.HasPrefix(link, "https://www.youtube.com/channel") || strings.HasPrefix(link, "https://youtube.com/channel") {
		return ""
	}

	return link
}

// CleanReleaseTitle очищает название релиза
func (u *ReleaseUtils) CleanReleaseTitle(title string) string {
	// Удаляем лишние пробелы
	title = strings.TrimSpace(title)

	// Удаляем специальные символы в начале и конце (НЕ удаляем квадратные скобки!)
	title = strings.Trim(title, "{}")

	// Заменяем множественные пробелы на один
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	return title
}

// CleanArtistName очищает имя артиста
func (u *ReleaseUtils) CleanArtistName(artist string) string {
	// Удаляем лишние пробелы
	artist = strings.TrimSpace(artist)

	// Удаляем специальные символы в начале и конце
	artist = strings.Trim(artist, "[](){}")

	// Заменяем множественные пробелы на один
	artist = regexp.MustCompile(`\s+`).ReplaceAllString(artist, " ")

	return artist
}

// ValidateRelease проверяет валидность релиза
func (u *ReleaseUtils) ValidateRelease(release *Release) error {
	if release.ArtistID <= 0 {
		return fmt.Errorf("artist_id is required")
	}
	if release.Date == "" {
		return fmt.Errorf("date is required")
	}

	return nil
}

// FormatReleaseForDisplay форматирует релиз для отображения
func (u *ReleaseUtils) FormatReleaseForDisplay(release *Release) string {
	var parts []string

	// Артист
	var artistName string
	if release.Artist != nil {
		artistName = release.Artist.Name
	}
	parts = append(parts, fmt.Sprintf("🎤 %s", artistName))

	// Название
	title := release.GetDisplayTitle()
	parts = append(parts, fmt.Sprintf("💿 %s", title))

	// Титульный трек
	if release.TitleTrack != "" && release.TitleTrack != "N/A" {
		parts = append(parts, fmt.Sprintf("🎵 %s", release.TitleTrack))
	}

	// Дата и время
	dateTime := release.GetFormattedDateTime()
	parts = append(parts, fmt.Sprintf("📅 %s", dateTime))

	// Тип релиза (упрощенный)
	typeEmoji := "🎵"
	parts = append(parts, fmt.Sprintf("%s Release", typeEmoji))

	// MV
	if release.HasMV() {
		parts = append(parts, fmt.Sprintf("🎬 [MV](%s)", release.MV))
	}

	return strings.Join(parts, "\n")
}

// FormatReleaseForTelegram форматирует релиз для Telegram сообщения
func (u *ReleaseUtils) FormatReleaseForTelegram(release *Release) string {
	var artistName string
	if release.Artist != nil {
		artistName = release.Artist.Name
	}
	artist := escapeHTML(artistName)
	albumName := escapeHTML(release.AlbumName)
	albumName = strings.TrimPrefix(albumName, "Album: ")
	albumName = strings.TrimPrefix(albumName, "OST: ")
	cleanedTitleTrack := strings.ReplaceAll(release.TitleTrack, "Title Track:", "")
	cleanedTitleTrack = strings.TrimSpace(cleanedTitleTrack)
	trackName := escapeHTML(cleanedTitleTrack)

	result := fmt.Sprintf("%s | <b>%s</b>", release.Date, artist)
	if albumName != "" && albumName != "N/A" {
		result += fmt.Sprintf(" | %s", albumName)
	}
	if release.MV != "" && release.MV != "N/A" {
		if trackName != "" && trackName != "N/A" {
			result += fmt.Sprintf(" | <a href=\"%s\">%s</a>", release.MV, trackName)
		} else {
			result += fmt.Sprintf(" | <a href=\"%s\">Link</a>", release.MV)
		}
	} else if trackName != "" && trackName != "N/A" {
		result += fmt.Sprintf(" | %s", trackName)
	}
	return result
}

// FormatDateWithYear форматирует дату в формате DD.MM.YY
func FormatDateWithYear(dateStr string, year string, logger interface{}) (string, error) {
	// Парсим дату в формате DD.MM.YY (например, "30.07.25")
	if !strings.Contains(dateStr, ".") || len(strings.Split(dateStr, ".")) != 3 {
		return "", fmt.Errorf("invalid date format, expected DD.MM.YY: %s", dateStr)
	}

	parts := strings.Split(dateStr, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid date format, expected DD.MM.YY: %s", dateStr)
	}

	day := parts[0]
	month := parts[1]
	yearFromDate := parts[2]

	// Если год в дате двухзначный, используем его, иначе используем переданный год
	var fullYear string
	if len(yearFromDate) == 2 {
		yearInt, err := strconv.Atoi(yearFromDate)
		if err != nil {
			return "", fmt.Errorf("failed to parse year from date: %w", err)
		}
		// Предполагаем, что годы 00-30 это 2000-2030, а 31-99 это 1931-1999
		if yearInt <= 30 {
			fullYear = fmt.Sprintf("20%s", yearFromDate)
		} else {
			fullYear = fmt.Sprintf("19%s", yearFromDate)
		}
	} else {
		fullYear = yearFromDate
	}

	// Формируем дату в формате "2006-01-02"
	dateStr = fmt.Sprintf("%s-%s-%s", fullYear, month, day)
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse date: %w", err)
	}

	// Форматируем в DD.MM.YY
	return parsedDate.Format("02.01.06"), nil
}
