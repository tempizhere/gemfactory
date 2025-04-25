package releasefmt

import (
	"fmt"
	"strings"
	"time"

	"gemfactory/internal/features/releasesbot/html"
	"gemfactory/internal/features/releasesbot/release"
	"go.uber.org/zap"
)

// FormatDate parses and formats a date string
func FormatDate(dateStr string, logger *zap.Logger) (string, error) {
	if dateStr == "" {
		logger.Debug("Empty date string")
		return "", fmt.Errorf("empty date string")
	}

	dateStr = strings.ReplaceAll(dateStr, ":", "")
	dateStr = strings.TrimSpace(dateStr)

	isDate := false
	for _, month := range release.Months {
		if strings.HasPrefix(strings.ToLower(dateStr), month) {
			isDate = true
			break
		}
	}
	if !isDate {
		logger.Debug("Not a valid date string", zap.String("date", dateStr))
		return "", fmt.Errorf("invalid date string: %s", dateStr)
	}

	var parsedDate time.Time
	var err error
	if strings.Contains(dateStr, ",") {
		parsedDate, err = time.Parse(release.DateParseFormat, dateStr)
	} else {
		parts := strings.Fields(dateStr)
		if len(parts) < 2 {
			logger.Debug("Invalid date format", zap.String("date", dateStr))
			return "", fmt.Errorf("invalid date format: %s", dateStr)
		}
		if len(parts) > 2 && strings.Contains(parts[len(parts)-1], "20") {
			dateStr = strings.Join(parts, " ")
			parsedDate, err = time.Parse("January 2 2006", dateStr)
		} else {
			dateStr = strings.Join(parts[:2], " ") + " " + release.CurrentYear()
			parsedDate, err = time.Parse("January 2 2006", dateStr)
		}
	}
	if err != nil {
		logger.Debug("Failed to parse date", zap.String("date", dateStr), zap.Error(err))
		return "", fmt.Errorf("failed to parse date '%s': %v", dateStr, err)
	}

	return parsedDate.Format(release.DateFormat), nil
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
			logger.Debug("No time after 'at'", zap.String("time", rawTime))
			return "", fmt.Errorf("invalid time format: %s", rawTime)
		}
	}

	parsedTime, err := time.Parse(release.TimeParseFormat, rawTime)
	if err != nil {
		parsedTime, err = time.Parse("3:04 PM", rawTime)
		if err != nil {
			logger.Debug("Failed to parse time", zap.String("time", rawTime), zap.Error(err))
			return "", fmt.Errorf("failed to parse time '%s': %v", rawTime, err)
		}
	}

	return parsedTime.Format(release.TimeFormat), nil
}

// ConvertKSTtoMSK converts KST time to MSK
func ConvertKSTtoMSK(kstTime string, logger *zap.Logger) (string, error) {
	if kstTime == "" {
		logger.Debug("Empty KST time")
		return "", fmt.Errorf("empty KST time")
	}

	parsedTime, err := time.Parse(release.TimeFormat, kstTime)
	if err != nil {
		logger.Debug("Failed to parse KST time", zap.String("time", kstTime), zap.Error(err))
		return "", fmt.Errorf("failed to parse KST time '%s': %v", kstTime, err)
	}

	mskTime := parsedTime.Add(release.KSTToMSKDiff)
	return mskTime.Format(release.TimeFormat), nil
}

// CleanLink cleans a YouTube link
func CleanLink(link string, logger *zap.Logger) string {
	if link == "" {
		logger.Debug("Empty link")
		return ""
	}
	if strings.Contains(link, "youtube.com/@") {
		logger.Debug("Link is a channel", zap.String("link", link))
		return ""
	}
	cleaned := strings.Split(link, "?")[0]
	logger.Debug("Cleaned link", zap.String("original", link), zap.String("cleaned", cleaned))
	return cleaned
}

// FormatReleaseForTelegram formats a release for Telegram
func FormatReleaseForTelegram(release release.Release, logger *zap.Logger) string {
	artist := html.Escape(release.Artist)
	albumName := html.Escape(release.AlbumName)
	albumName = strings.TrimPrefix(albumName, "Album: ")
	albumName = strings.TrimPrefix(albumName, "OST: ")
	cleanedTitleTrack := strings.ReplaceAll(release.TitleTrack, "Title Track:", "")
	cleanedTitleTrack = strings.TrimSpace(cleanedTitleTrack)
	trackName := html.Escape(cleanedTitleTrack)

	result := fmt.Sprintf("%s | <b>%s</b>", release.Date, artist)
	if albumName != "N/A" { // Отображаем альбом, если он есть
		result += fmt.Sprintf(" | %s", albumName)
	}
	if release.MV != "" && release.MV != "N/A" {
		if trackName != "N/A" {
			result += fmt.Sprintf(" | <a href=\"%s\">%s</a>", release.MV, trackName)
		} else {
			result += fmt.Sprintf(" | <a href=\"%s\">Link</a>", release.MV)
		}
	} else if trackName != "N/A" {
		result += fmt.Sprintf(" | %s", trackName)
	}
	logger.Debug("Formatted release for Telegram", zap.String("release", result))
	return result
}
