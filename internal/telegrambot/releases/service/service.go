package service

import (
	"fmt"
	"gemfactory/internal/telegrambot/releases/release"
	"html"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// escapeHTML escapes special HTML characters in a string
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// dateCache holds cached parsed dates
var dateCache sync.Map

// FormatDate parses and formats a date string
func FormatDate(dateStr string, logger *zap.Logger) (string, error) {
	if dateStr == "" {
		logger.Debug("Empty date string")
		return "", fmt.Errorf("empty date string")
	}

	dateStr = strings.ReplaceAll(dateStr, ":", "")
	dateStr = strings.TrimSpace(dateStr)

	// Check cache
	if cached, ok := dateCache.Load(dateStr); ok {
		return cached.(string), nil
	}

	parsedDate, err := parseDate(dateStr, logger)
	if err != nil {
		return "", err
	}

	formatted := parsedDate.Format("02.01.06")
	dateCache.Store(dateStr, formatted)
	return formatted, nil
}

// parseDate parses a date string with or without a year
func parseDate(dateStr string, logger *zap.Logger) (time.Time, error) {
	cfg := release.NewConfig()
	isDate := false
	for _, month := range cfg.Months() {
		if strings.HasPrefix(strings.ToLower(dateStr), month) {
			isDate = true
			break
		}
	}
	if !isDate {
		logger.Debug("Invalid date string", zap.String("date", dateStr))
		return time.Time{}, fmt.Errorf("invalid date string: %s", dateStr)
	}

	if strings.Contains(dateStr, ",") {
		return parseDateWithComma(dateStr, logger)
	}

	parts := strings.Fields(dateStr)
	if len(parts) < 2 {
		logger.Debug("Invalid date format", zap.String("date", dateStr))
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
	}

	if len(parts) > 2 && strings.Contains(parts[len(parts)-1], "20") {
		// Полный формат с годом, например, "January 2 2023"
		dateStr = strings.Join(parts, " ")
		parsedDate, err := time.Parse("January 2 2006", dateStr)
		if err != nil {
			logger.Debug("Failed to parse date", zap.String("date", dateStr), zap.Error(err))
			return time.Time{}, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
		}
		return parsedDate, nil
	}

	// Формат без года, добавляем текущий год
	dateStr = strings.Join(parts[:2], " ") + " " + release.CurrentYear()
	parsedDate, err := time.Parse("January 2 2006", dateStr)
	if err != nil {
		logger.Debug("Failed to parse date", zap.String("date", dateStr), zap.Error(err))
		return time.Time{}, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
	}
	return parsedDate, nil
}

// parseDateWithComma parses a date string with commas
func parseDateWithComma(dateStr string, logger *zap.Logger) (time.Time, error) {
	cfg := release.NewConfig()
	parsedDate, err := time.Parse(cfg.DateParseFormat(), dateStr)
	if err != nil {
		logger.Debug("Failed to parse date with comma", zap.String("date", dateStr), zap.Error(err))
		return time.Time{}, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
	}
	return parsedDate, nil
}

// FormatTimeKST parses KST time and returns it in 24-hour format
func FormatTimeKST(rawTime string, logger *zap.Logger) (string, error) {
	if rawTime == "" {
		logger.Debug("Empty time string")
		return "", fmt.Errorf("empty time string")
	}

	rawTime = strings.ReplaceAll(rawTime, "KST", "")
	rawTime = strings.TrimSpace(rawTime)
	if strings.Contains(rawTime, "at") {
		parts := strings.Split(rawTime, "at")
		if len(parts) > 1 {
			rawTime = strings.TrimSpace(parts[1])
		} else {
			logger.Debug("Invalid time format", zap.String("time", rawTime))
			return "", fmt.Errorf("invalid time format: %s", rawTime)
		}
	}

	cfg := release.NewConfig()
	parsedTime, err := time.Parse(cfg.TimeParseFormat(), rawTime)
	if err != nil {
		parsedTime, err = time.Parse("3:04 PM", rawTime)
		if err != nil {
			logger.Debug("Failed to parse time", zap.String("time", rawTime), zap.Error(err))
			return "", fmt.Errorf("failed to parse time '%s': %w", rawTime, err)
		}
	}

	return parsedTime.Format(cfg.TimeFormat()), nil
}

// ConvertKSTtoMSK converts KST time to MSK
func ConvertKSTtoMSK(kstTime string, logger *zap.Logger) (string, error) {
	if kstTime == "" {
		logger.Debug("Empty KST time")
		return "", fmt.Errorf("empty KST time")
	}

	cfg := release.NewConfig()
	parsedTime, err := time.Parse(cfg.TimeFormat(), kstTime)
	if err != nil {
		logger.Debug("Failed to parse KST time", zap.String("time", kstTime), zap.Error(err))
		return "", fmt.Errorf("failed to parse KST time '%s': %w", kstTime, err)
	}

	mskTime := parsedTime.Add(cfg.KSTToMSKDiff())
	return mskTime.Format(cfg.TimeFormat()), nil
}

// CleanLink cleans a YouTube link
func CleanLink(link string, logger *zap.Logger) string {
	if link == "" {
		logger.Debug("Empty link")
		return ""
	}

	if strings.HasPrefix(link, "https://www.youtube.com/@") || strings.HasPrefix(link, "https://youtube.com/@") ||
		strings.HasPrefix(link, "https://www.youtube.com/channel") || strings.HasPrefix(link, "https://youtube.com/channel") {
		logger.Debug("Link is a channel link", zap.String("link", link))
		return ""
	}

	return link
}

// FormatReleaseForTelegram formats a release for Telegram message
func FormatReleaseForTelegram(rel release.Release) string {
	artist := escapeHTML(rel.Artist)
	albumName := escapeHTML(rel.AlbumName)
	albumName = strings.TrimPrefix(albumName, "Album: ")
	albumName = strings.TrimPrefix(albumName, "OST: ")
	cleanedTitleTrack := strings.ReplaceAll(rel.TitleTrack, "Title Track:", "")
	cleanedTitleTrack = strings.TrimSpace(cleanedTitleTrack)
	trackName := escapeHTML(cleanedTitleTrack)

	result := fmt.Sprintf("%s | <b>%s</b>", rel.Date, artist)
	if albumName != "" && albumName != "N/A" {
		result += fmt.Sprintf(" | %s", albumName)
	}
	if rel.MV != "" && rel.MV != "N/A" {
		if trackName != "" && trackName != "N/A" {
			result += fmt.Sprintf(" | <a href=\"%s\">%s</a>", rel.MV, trackName)
		} else {
			result += fmt.Sprintf(" | <a href=\"%s\">Link</a>", rel.MV)
		}
	} else if trackName != "" && trackName != "N/A" {
		result += fmt.Sprintf(" | %s", trackName)
	}
	return result
}
