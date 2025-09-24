// Package scraper содержит вспомогательные функции для веб-скрапинга.
package scraper

import (
	"gemfactory/internal/model"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// CurrentYear возвращает текущий год как строку
func CurrentYear() string {
	return time.Now().Format("2006")
}

// DecodeHTMLEntities декодирует HTML entities в строке
func DecodeHTMLEntities(text string) string {
	// Сначала используем стандартную функцию html.UnescapeString
	decoded := html.UnescapeString(text)

	// Дополнительно декодируем специфичные HTML entities
	decoded = strings.ReplaceAll(decoded, "&nbsp;", " ")
	decoded = strings.ReplaceAll(decoded, "&#160;", " ")
	decoded = strings.ReplaceAll(decoded, "&#32;", " ")
	decoded = strings.ReplaceAll(decoded, "&lt;", "<")
	decoded = strings.ReplaceAll(decoded, "&gt;", ">")
	decoded = strings.ReplaceAll(decoded, "&amp;", "&")
	decoded = strings.ReplaceAll(decoded, "&quot;", "\"")
	decoded = strings.ReplaceAll(decoded, "&#39;", "'")

	// Декодируем числовые HTML entities для кавычек и тире
	decoded = strings.ReplaceAll(decoded, "&#8211;", "–")  // En dash
	decoded = strings.ReplaceAll(decoded, "&#8216;", "'")  // Left single quotation mark
	decoded = strings.ReplaceAll(decoded, "&#8217;", "'")  // Right single quotation mark
	decoded = strings.ReplaceAll(decoded, "&#8220;", "\"") // Left double quotation mark
	decoded = strings.ReplaceAll(decoded, "&#8221;", "\"") // Right double quotation mark
	decoded = strings.ReplaceAll(decoded, "&#8212;", "—")  // Em dash
	decoded = strings.ReplaceAll(decoded, "&#8230;", "…")  // Horizontal ellipsis

	// Заменяем различные типы Unicode пробелов на обычные пробелы
	decoded = strings.ReplaceAll(decoded, "\u00A0", " ") // Non-breaking space
	decoded = strings.ReplaceAll(decoded, "\u2000", " ") // En quad
	decoded = strings.ReplaceAll(decoded, "\u2001", " ") // Em quad
	decoded = strings.ReplaceAll(decoded, "\u2002", " ") // En space
	decoded = strings.ReplaceAll(decoded, "\u2003", " ") // Em space
	decoded = strings.ReplaceAll(decoded, "\u2004", " ") // Three-per-em space
	decoded = strings.ReplaceAll(decoded, "\u2005", " ") // Four-per-em space
	decoded = strings.ReplaceAll(decoded, "\u2006", " ") // Six-per-em space
	decoded = strings.ReplaceAll(decoded, "\u2007", " ") // Figure space
	decoded = strings.ReplaceAll(decoded, "\u2008", " ") // Punctuation space
	decoded = strings.ReplaceAll(decoded, "\u2009", " ") // Thin space
	decoded = strings.ReplaceAll(decoded, "\u200A", " ") // Hair space
	decoded = strings.ReplaceAll(decoded, "\u202F", " ") // Narrow no-break space
	decoded = strings.ReplaceAll(decoded, "\u205F", " ") // Medium mathematical space
	decoded = strings.ReplaceAll(decoded, "\u3000", " ") // Ideographic space

	return decoded
}

// ExtractTrackFromQuotes извлекает название трека из кавычек с правильной обработкой HTML entities
func ExtractTrackFromQuotes(text string, logger *zap.Logger) string {
	// Сначала декодируем HTML entities
	decoded := DecodeHTMLEntities(text)

	// Убираем тире перед кавычками, чтобы они не мешали поиску
	decoded = strings.ReplaceAll(decoded, "–", "")
	decoded = strings.ReplaceAll(decoded, "-", "")

	// Используем регулярные выражения для поиска треков в кавычках
	// Ищем одинарные кавычки (включая Unicode)
	singleQuoteRegex := regexp.MustCompile(`['']([^'']+)['']`)
	matches := singleQuoteRegex.FindAllStringSubmatch(decoded, -1)
	for _, match := range matches {
		if len(match) > 1 {
			track := strings.TrimSpace(match[1])
			// Убираем лишние символы в начале и конце
			track = strings.TrimPrefix(track, "–")
			track = strings.TrimPrefix(track, "-")
			track = strings.TrimSpace(track)
			// КРИТИЧНО: Применяем cleanTrackName для удаления всех артефактов
			track = cleanTrackName(track)
			if len(track) > 0 && isValidUnifiedTrack(track) {
				return track
			}
		}
	}

	// Ищем двойные кавычки (включая Unicode)
	doubleQuoteRegex := regexp.MustCompile(`[""]([^""]+)[""]`)
	matches = doubleQuoteRegex.FindAllStringSubmatch(decoded, -1)
	for _, match := range matches {
		if len(match) > 1 {
			track := strings.TrimSpace(match[1])
			// Убираем лишние символы в начале и конце
			track = strings.TrimPrefix(track, "–")
			track = strings.TrimPrefix(track, "-")
			track = strings.TrimSpace(track)
			// КРИТИЧНО: Применяем cleanTrackName для удаления всех артефактов
			track = cleanTrackName(track)
			if len(track) > 0 && isValidUnifiedTrack(track) {
				return track
			}
		}
	}

	// Ищем угловые скобки
	angleBracketRegex := regexp.MustCompile(`<([^>]+)>`)
	matches = angleBracketRegex.FindAllStringSubmatch(decoded, -1)
	for _, match := range matches {
		if len(match) > 1 {
			track := strings.TrimSpace(match[1])
			// КРИТИЧНО: Применяем cleanTrackName для удаления всех артефактов
			track = cleanTrackName(track)
			if len(track) > 1 && isValidUnifiedTrack(track) {
				return track
			}
		}
	}

	// Если не нашли трек в кавычках, попробуем найти по YouTube ссылке
	if strings.Contains(strings.ToLower(decoded), "youtube") {
		// Ищем строки с YouTube и пытаемся извлечь трек из предыдущих строк
		lines := strings.Split(decoded, "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), "youtube") {
				// Ищем трек в предыдущих строках
				for j := i - 1; j >= 0 && j >= i-3; j-- {
					if j < len(lines) {
						prevLine := strings.TrimSpace(lines[j])
						if prevLine != "" {
							// Пробуем извлечь трек из предыдущей строки
							track := extractTrackFromLine(prevLine, logger)
							track = cleanTrackName(track) // КРИТИЧНО: Очищаем артефакты
							if track != "" && isValidUnifiedTrack(track) {
								return track
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// FallbackTrackExtraction извлекает трек из сложных случаев, когда основной парсер не справляется
func FallbackTrackExtraction(text string, logger *zap.Logger) string {
	logger.Info("FallbackTrackExtraction called", zap.String("text", text))

	// Убираем тире и пробелы
	cleaned := strings.ReplaceAll(text, "–", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.TrimSpace(cleaned)

	logger.Info("Fallback: cleaned text", zap.String("cleaned", cleaned))

	// УНИВЕРСАЛЬНАЯ ЛОГИКА: Проверяем, есть ли множественные релизы
	// Ищем любые даты в любом формате
	datePattern := regexp.MustCompile(`(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d+.*?:\s*`)
	matches := datePattern.FindAllStringSubmatch(cleaned, -1)

	if len(matches) > 1 {
		logger.Info("Fallback: detected multiple releases pattern", zap.Int("date_count", len(matches)))

		// Берем первый трек из первого блока даты
		if len(matches) > 0 {
			dateStr := matches[0][0]
			dateIndex := strings.Index(cleaned, dateStr)
			if dateIndex != -1 {
				// Берем текст после первой даты до следующей даты
				afterDate := cleaned[dateIndex+len(dateStr):]

				// Ищем следующую дату
				nextDateIndex := -1
				allMonths := []string{"January", "February", "March", "April", "May", "June",
					"July", "August", "September", "October", "November", "December"}
				for _, month := range allMonths {
					if idx := strings.Index(afterDate, month); idx != -1 {
						if nextDateIndex == -1 || idx < nextDateIndex {
							nextDateIndex = idx
						}
					}
				}
				if nextDateIndex != -1 {
					afterDate = afterDate[:nextDateIndex]
				}

				afterDate = strings.TrimSpace(afterDate)
				logger.Info("Fallback: extracted first date block", zap.String("block", afterDate))

				// Пробуем извлечь трек из этого блока
				track := extractTrackFromDateBlock(afterDate, logger)
				if track != "" {
					logger.Info("Fallback: SUCCESS with multiple releases pattern", zap.String("track", track))
					return track
				}
			}
		}
	}

	// ПРИОРИТЕТ: Детекция структуры по содержимому (ПЕРЕД старым кодом)
	logger.Info("Fallback: detecting structure type")
	structType := detectStructureTypeByContent(cleaned)

	// УНИФИЦИРОВАННАЯ ОБРАБОТКА (для всех случаев)
	track := extractTrackUnified(cleaned, logger)
	if track != "" {
		logger.Info("Fallback: SUCCESS with unified extraction",
			zap.String("track", track),
			zap.String("type", structType.String()))
		return track
	}
	logger.Info("Fallback: unified extraction failed, trying fallbacks", zap.String("type", structType.String()))

	// ДОПОЛНИТЕЛЬНАЯ ЛОГИКА: Пробуем извлечь трек из текущего блока напрямую (только для простых случаев)
	logger.Info("Fallback: trying direct extraction from current block")
	track = extractTrackFromDateBlock(cleaned, logger)
	if track != "" {
		logger.Info("Fallback: SUCCESS with direct extraction", zap.String("track", track))
		return track
	}

	// Fallback: пытаемся извлечь трек из множественных дат
	logger.Info("Fallback: trying multiple dates extraction")
	track = extractTrackFromMultipleDates(cleaned, logger)
	if track != "" {
		logger.Info("Fallback: SUCCESS with multiple dates extraction", zap.String("track", track))
		return track
	}

	// СПЕЦИАЛЬНАЯ ОБРАБОТКА: Для MONSTA X и подобных случаев
	if strings.Contains(cleaned, "Prerelease") || strings.Contains(cleaned, "Pre-release") {
		logger.Info("Fallback: detected Prerelease pattern, trying special extraction")

		// ПРОСТОЙ ПАТТЕРН: "Do What I Want" (любой текст после) - БЕЗ ПРОВЕРКИ isValidTrackName
		if match := regexp.MustCompile(`[""]([^""]+)[""]`).FindStringSubmatch(cleaned); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if len(track) > 0 {
				logger.Info("Fallback: SUCCESS with simple quotes in Prerelease", zap.String("track", track))
				return track
			}
		}

		// Паттерн: August 18, 2025: Prerelease "Do What I Want"
		if match := regexp.MustCompile(`\w+\s+\d+,\s+\d+:\s+Pre-?release\s+[""]([^""]+)[""]`).FindStringSubmatch(cleaned); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				logger.Info("Fallback: SUCCESS with Prerelease pattern", zap.String("track", track))
				return track
			}
		}

		// Паттерн: Prerelease "Do What I Want" (без даты)
		if match := regexp.MustCompile(`Pre-?release\s+[""]([^""]+)[""]`).FindStringSubmatch(cleaned); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				logger.Info("Fallback: SUCCESS with simple Prerelease pattern", zap.String("track", track))
				return track
			}
		}
	}

	// НОВАЯ СИСТЕМА: Context-Aware Parser
	// Анализируем каждую строку отдельно для лучшего понимания контекста
	lines := strings.Split(cleaned, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Паттерн 1: Дата: "TRACK" Release/MV (более гибкий)
		if match := regexp.MustCompile(`\w+\s+\d+:\s*[""]([^""]+)[""]\s+(?:Release|MV|M/V|Prerelease)`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 1.1: Более простой - любая строка с датой и кавычками
		if match := regexp.MustCompile(`\w+\s+\d+.*[""]([^""]+)[""]`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 2: Title Trac: 'TRACK' (с опечаткой)
		if match := regexp.MustCompile(`Title\s+Trac:\s*['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				logger.Info("Fallback: found track with Title Trac pattern", zap.String("track", track), zap.String("line", line))
				return track
			}
		}

		// Паттерн 3: Single 'TRACK'
		if match := regexp.MustCompile(`\w+\s+Single\s+['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				logger.Info("Fallback: found track with Single pattern", zap.String("track", track), zap.String("line", line))
				return track
			}
		}

		// Паттерн 4: Artist x Artist "TRACK" Release
		if match := regexp.MustCompile(`\w+\s+x\s+\w+.*[""]([^""]+)[""]\s+Release`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				logger.Info("Fallback: found track with collaboration pattern", zap.String("track", track), zap.String("line", line))
				return track
			}
		}

		// Паттерн 5: 'TRACK' MV release (с тире)
		if match := regexp.MustCompile(`['']([^'']+)['']\s+MV\s+release`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				logger.Info("Fallback: found track with MV release pattern", zap.String("track", track), zap.String("line", line))
				return track
			}
		}

		// Паттерн 5.1: – 'TRACK' MV Release (с тире в начале)
		if match := regexp.MustCompile(`['']([^'']+)['']\s+MV\s+Release`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				logger.Info("Fallback: found track with MV Release pattern (with dash)", zap.String("track", track), zap.String("line", line))
				return track
			}
		}

		// Паттерн 6: "TRACK" Prerelease
		if match := regexp.MustCompile(`[""]([^""]+)[""]\s+Prerelease`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				logger.Info("Fallback: found track with Prerelease pattern", zap.String("track", track), zap.String("line", line))
				return track
			}
		}

		// Паттерн 6.1: "TRACK" (Title Track)
		if match := regexp.MustCompile(`[""]([^""]+)[""]\s+\(Title\s+Track\)`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 6.2: "TRACK" feat. ARTIST
		if match := regexp.MustCompile(`[""]([^""]+)[""]\s+feat\.`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 10: Digital Single – TRACK: (более гибкий)
		if match := regexp.MustCompile(`Digital\s+Single\s*[–-]\s*([^:\n]+):?`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 11: Pre-Release 'TRACK' (более гибкий)
		if match := regexp.MustCompile(`Pre-?Release:?\s*['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 12: Special Digital Single "TRACK" (более гибкий)
		if match := regexp.MustCompile(`Special\s+Digital\s+Single\s+[""]([^""]+)[""]`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 13: Digital Single: 'TRACK' (более гибкий)
		if match := regexp.MustCompile(`Digital\s+Single:\s*['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 14: Digital Single 'TRACK' (более гибкий)
		if match := regexp.MustCompile(`Digital\s+Single\s+['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 15: Special Digital Single 'TRACK' (более гибкий)
		if match := regexp.MustCompile(`\d+st?\s+Special\s+Digital\s+Single\s+['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 16: Title Track: – TRACK (более гибкий)
		if match := regexp.MustCompile(`Title\s+Track:\s*[–-]\s*([^\n]+)`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 17: дата + Pre-Release 'TRACK' (более гибкий)
		if match := regexp.MustCompile(`\w+\s+\d+:\s+Pre-?Release\s+['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 18: японские кавычки 「TRACK」 (более гибкий)
		if match := regexp.MustCompile(`「([^」]+)」`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 19: SG 'TRACK' (более гибкий)
		if match := regexp.MustCompile(`\d+rd?\s+SG\s+['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 20: трек в скобках без кавычек (для японских названий) (более гибкий)
		if match := regexp.MustCompile(`([^()]+)\s*\(([^)]+)\)`).FindStringSubmatch(line); len(match) > 2 {
			// Проверяем, что это не альбом и не служебная информация
			if !strings.Contains(strings.ToLower(line), "album") &&
				!strings.Contains(strings.ToLower(line), "teaser") &&
				!strings.Contains(strings.ToLower(line), "poster") {
				track := strings.TrimSpace(match[1])
				if isValidTrackName(track) {
					return track
				}
			}
		}

		// Паттерн 21: Single『TRACK』 (японские кавычки для Single)
		if match := regexp.MustCompile(`\d+rd?\s+Single『([^』]+)』`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 22: трек без кавычек (только японские символы)
		if match := regexp.MustCompile(`^([^\s]+)\s*\([^)]+\)$`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			// Проверяем, что это японские символы
			if regexp.MustCompile(`[\p{Hiragana}\p{Katakana}\p{Han}]`).MatchString(track) && isValidTrackName(track) {
				return track
			}
		}

		// Паттерн 7: [квадратные скобки] - только если это не альбом
		if match := regexp.MustCompile(`\[([^\]]+)\]`).FindStringSubmatch(line); len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if isValidTrackName(track) && !strings.Contains(strings.ToLower(line), "album") {
				logger.Info("Fallback: found track with square brackets pattern", zap.String("track", track), zap.String("line", line))
				return track
			}
		}
	}

	// Если не нашли по контексту, пробуем общие паттерны
	// Паттерн 8: Любые Unicode одинарные кавычки (но не в контексте альбома)
	if match := regexp.MustCompile(`['']([^'']+)['']`).FindStringSubmatch(cleaned); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) && !strings.Contains(strings.ToLower(cleaned), "album") {
			return track
		}
	}

	// Паттерн 9: Любые Unicode двойные кавычки (но не в контексте альбома)
	if match := regexp.MustCompile(`[""]([^""]+)[""]`).FindStringSubmatch(cleaned); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) && !strings.Contains(strings.ToLower(cleaned), "album") {
			return track
		}
	}

	// ПОСЛЕДНИЙ FALLBACK: Используем старую простую логику
	logger.Info("Fallback: trying old-style extraction")

	// Преобразуем текст в строки для старой логики
	eventLines := strings.Split(cleaned, "\n")
	oldStyleTrack := ExtractTrackNameOldStyle(eventLines, logger)
	if oldStyleTrack != "" && oldStyleTrack != "N/A" {
		logger.Info("Fallback: SUCCESS with old-style extraction", zap.String("track", oldStyleTrack))
		return oldStyleTrack
	}

	// Логируем все случаи, где даже старая логика не сработала
	logger.Info("Fallback: FAILED TO EXTRACT TRACK",
		zap.String("original_text", text),
		zap.String("cleaned_text", cleaned),
		zap.String("analysis", "NO PATTERNS MATCHED - EVEN OLD STYLE FAILED"))

	return ""
}

// extractTrackFromMultipleDates извлекает трек из строк с множественными датами
// Обрабатывает случаи типа "January 25: "ZEN" Release, January 31: "Love Hangover" Release"
func extractTrackFromMultipleDates(text string, logger *zap.Logger) string {
	logger.Info("Extracting track from multiple dates", zap.String("text", text))

	// НОВЫЕ UNICODE-AWARE ПАТТЕРНЫ
	patterns := []string{
		// Основные паттерны с Unicode кавычками для дат
		`(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d+:\s*[""]([^""]+)[""]`,
		`(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d+:\s*['']([^'']+)['']`,

		// Паттерны с сокращенными месяцами
		`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d+:\s*[""]([^""]+)[""]`,
		`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d+:\s*['']([^'']+)['']`,

		// Паттерны без дат (для ARTMS и подобных)
		`['']([^'']+)[''](?:\s+(?:MV|Release|Audio|Teaser))`,
		`[""]([^""]+)[""](?:\s+(?:MV|Release|Audio|Teaser))`,

		// Специальные паттерны для Title Track
		`Title\s+Track:\s*[""]([^""]+)[""]`,
		`Title\s+Track:\s*['']([^'']+)['']`,

		// Паттерны для начала строки (ARTMS случай)
		`^[^''"]*['']([^'']+)['']`,
		`^[^"]*[""]([^""]+)[""]`,

		// ASCII кавычки как fallback
		`(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d+:\s*["']([^"']+)["']`,
		`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d+:\s*["']([^"']+)["']`,
		`\d+\s+(January|February|March|April|May|June|July|August|September|October|November|December):\s*["']([^"']+)["']`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)

		if len(matches) > 0 {
			var track string
			// Определяем индекс группы с треком
			if len(matches[0]) > 2 {
				track = matches[0][2] // Вторая группа (после месяца)
			} else if len(matches[0]) > 1 {
				track = matches[0][1] // Первая группа
			}

			track = strings.TrimSpace(track)
			track = cleanTrackName(track) // Новая функция очистки

			if track != "" && len(track) > 1 {
				logger.Info("Found track from multiple dates",
					zap.String("track", track),
					zap.String("pattern", pattern))
				return track
			}
		}
	}

	// Если не нашли по основным паттернам, ищем любые треки в кавычках
	fallbackPatterns := []string{
		// Unicode кавычки
		`^[""]([^""]+)[""]`,                           // Начало строки
		`[""]([^""]+)[""](?:\s+(?:Release|MV|Audio))`, // Трек + ключевое слово
		`^['']([^'']+)['']`,                           // Начало строки (одинарные)
		`['']([^'']+)[''](?:\s+(?:Release|MV|Audio))`, // Трек + ключевое слово

		// ASCII кавычки как fallback
		`^["']([^"']+)["']`,                           // Начало строки
		`["']([^"']+)["'](?:\s+(?:Release|MV|Audio))`, // Трек + ключевое слово
	}

	for _, pattern := range fallbackPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)

		if len(matches) > 1 {
			track := matches[1]
			track = strings.TrimSpace(track)
			track = cleanTrackName(track)

			if track != "" && len(track) > 1 {
				logger.Info("Found track from fallback pattern",
					zap.String("track", track),
					zap.String("pattern", pattern))
				return track
			}
		}
	}

	logger.Info("No track found in multiple dates")
	return ""
}

// cleanTrackName очищает название трека от артефактов парсинга
func cleanTrackName(track string) string {
	if track == "" {
		return ""
	}

	// Убираем артефакты парсинга в конце строки
	cleaners := []string{
		`\\" Music Video:.*`,
		`\\" MV.*`,
		`\\" Release.*`,
		`\\" Audio.*`,
		`" Music Video:.*`,
		`" MV.*`,
		`" Release.*`,
		`" Audio.*`,
		`\s+Music Video:.*`,
		`\s+MV.*`,
		`\s+Release.*`,
		`\s+Audio.*`,
		`\s+feat\..*`, // Убираем feat. и всё после

		// НОВЫЕ АРТЕФАКТЫ: Убираем в конце строки
		`\s+MV\s+Release$`,
		`\s+Special\s+Video$`,
		`\s+Official\s+Audio$`,
		`\s+Music\s+Video$`,
		`\s+Teaser$`,
		`\s+Release$`,
	}

	for _, cleaner := range cleaners {
		if re := regexp.MustCompile(cleaner); re != nil {
			track = re.ReplaceAllString(track, "")
		}
	}

	// Убираем лишние символы в конце
	track = strings.TrimSpace(track)
	track = strings.Trim(track, ".,;:") // НЕ удаляем закрывающие скобки!

	// ВАЖНО: НЕ удаляем квадратные скобки из названий альбомов!
	// Если трек заканчивается на ], это может быть часть названия альбома

	// Убираем экранирующие символы
	track = strings.ReplaceAll(track, `\"`, `"`)
	track = strings.ReplaceAll(track, `\'`, `'`)

	// Убираем тире в начале
	track = strings.TrimPrefix(track, "–")
	track = strings.TrimPrefix(track, "-")
	track = strings.TrimSpace(track)

	return track
}

// StructureType определяет тип структуры для специализированной обработки
type StructureType int

const (
	TypeSimple               StructureType = iota
	TypeMultipleDatesInBlock               // JENNIE - множественные даты в одном блоке
	TypeSingleTrackNoDate                  // ARTMS - трек без даты
	TypeKoreanTrack                        // KickFlip - корейские названия
	TypeCumulative                         // CORTIS - кумулятивная структура (уже исправлено)
)

// String возвращает строковое представление типа структуры
func (st StructureType) String() string {
	switch st {
	case TypeSimple:
		return "Simple"
	case TypeMultipleDatesInBlock:
		return "MultipleDatesInBlock"
	case TypeSingleTrackNoDate:
		return "SingleTrackNoDate"
	case TypeKoreanTrack:
		return "KoreanTrack"
	case TypeCumulative:
		return "Cumulative"
	default:
		return "Unknown"
	}
}

// detectStructureTypeByContent определяет тип структуры по содержимому
func detectStructureTypeByContent(content string) StructureType {
	content = strings.ToLower(content)

	// Проверяем на множественные даты с разными месяцами
	monthCount := 0
	months := []string{"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december"}
	for _, month := range months {
		if strings.Count(content, month) > 0 {
			monthCount++
		}
	}
	if monthCount > 1 {
		return TypeMultipleDatesInBlock
	}

	// Проверяем на POSTPONED и трек в кавычках
	if strings.Contains(content, "postponed") {
		return TypeSingleTrackNoDate
	}

	// Проверяем на корейские символы
	if containsKorean(content) {
		return TypeKoreanTrack
	}

	// Проверяем на кумулятивную структуру (множественные даты одного месяца)
	for _, month := range months {
		if strings.Count(content, month) > 1 {
			return TypeCumulative
		}
	}

	return TypeSimple
}

// containsKorean проверяет наличие корейских символов
func containsKorean(text string) bool {
	for _, r := range text {
		if (r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
			(r >= 0x1100 && r <= 0x11FF) || // Hangul Jamo
			(r >= 0x3130 && r <= 0x318F) { // Hangul Compatibility Jamo
			return true
		}
	}
	return false
}

// extractTrackUnified УНИФИЦИРОВАННЫЙ экстрактор для всех случаев
func extractTrackUnified(content string, logger *zap.Logger) string {
	logger.Info("Unified track extraction", zap.String("content", content[:min(100, len(content))]))

	// ПРОСТЫЕ И ЭФФЕКТИВНЫЕ ПАТТЕРНЫ в порядке приоритета
	patterns := []string{
		// 1. Title Track паттерны (ПРИОРИТЕТ)
		`Title\s+Track:\s*[""]([^""]+)[""]`,
		`Title\s+Track:\s*['']([^'']+)['']`,
		`Title\s+Track:\s*["']([^"']+)["']`,
		`Title\s+Track:\s*([^"\n]+)`, // Без кавычек

		// 2. Треки с MV Release (убираем артефакты)
		`[""]([^""]+?)[""]\s+MV\s+Release`,
		`['']([^'']+?)['']\s+MV\s+Release`,
		`["']([^"']+?)["']\s+MV\s+Release`,

		// 3. Треки с различными артефактами (убираем)
		`[""]([^""]+?)[""]\s+(?:Special\s+Video|Official\s+Audio|Music\s+Video|Teaser|Release)`,
		`['']([^'']+?)['']\s+(?:Special\s+Video|Official\s+Audio|Music\s+Video|Teaser|Release)`,
		`["']([^"']+?)["']\s+(?:Special\s+Video|Official\s+Audio|Music\s+Video|Teaser|Release)`,

		// 4. Даты с треками (любые месяцы и годы)
		`(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d+.*?:\s*[""]([^""]+)[""]`,
		`(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d+.*?:\s*['']([^'']+)['']`,
		`(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d+.*?:\s*["']([^"']+)["']`,

		// 6. Любые треки в кавычках (fallback)
		`[""]([^""]+)[""]`, // Unicode двойные кавычки
		`['']([^'']+)['']`, // Unicode одинарные кавычки
		`["']([^"']+)["']`, // ASCII кавычки

		// 7. POSTPONED треки (ARTMS)
		`POSTPONED[^)]*\)\s*['']([^'']+)['']`,
		`POSTPONED[^)]*\)\s*[""]([^""]+)[""]`,
		`POSTPONED[^)]*\)\s*["']([^"']+)["']`,

		// 8. Специальные паттерны для проблемных случаев
		`Pre-release\s+[""]([^""]+)[""]`,
		`Pre-release\s+['']([^'']+)['']`,
		`Pre-release\s+["']([^"']+)["']`,

		// 9. Паттерны для Digital Release
		`Digital\s+Release.*[""]([^""]+)[""]`,
		`Digital\s+Release.*['']([^'']+)['']`,
		`Digital\s+Release.*["']([^"']+)["']`,

		// 10. Паттерны для Album Physical Release
		`Album\s+Physical\s+Release.*[""]([^""]+)[""]`,
		`Album\s+Physical\s+Release.*['']([^'']+)['']`,
		`Album\s+Physical\s+Release.*["']([^"']+)["']`,

		// 11. Паттерны для случаев без явных треков - ищем в соседних строках
		`Album:\s*([^"\n]+?)(?:\s*–|\s*\[|\s*\(|$)`,         // Album: название альбома
		`EP\s*[:\-]\s*([^"\n]+?)(?:\s*–|\s*\[|\s*\(|$)`,     // EP: название
		`Single\s*[:\-]\s*([^"\n]+?)(?:\s*–|\s*\[|\s*\(|$)`, // Single: название

		// 12. Японские символы 『』
		`『([^』]+)』`,

		// 13. Первый трек в начале строки
		`^[^"']*?[""]([^""]+?)[""]`,
		`^[^"']*?['']([^'']+?)['']`,
		`^[^"']*?["']([^"']+?)["']`,
	}

	for i, pattern := range patterns {
		if re := regexp.MustCompile(pattern); re != nil {
			if matches := re.FindStringSubmatch(content); len(matches) > 1 {
				var track string
				// Определяем индекс группы с треком
				if len(matches) > 2 {
					track = strings.TrimSpace(matches[2]) // После месяца
				} else {
					track = strings.TrimSpace(matches[1]) // Основная группа
				}

				track = cleanTrackName(track)

				// ВАЛИДАЦИЯ: проверяем что это действительно трек
				if isValidUnifiedTrack(track) {
					logger.Info("Unified extraction SUCCESS",
						zap.String("track", track),
						zap.String("pattern", pattern),
						zap.Int("pattern_index", i))
					return track
				}
			}
		}
	}

	logger.Info("Unified extraction failed")
	return ""
}

// isValidUnifiedTrack проверяет валидность извлеченного трека
func isValidUnifiedTrack(track string) bool {
	if track == "" || len(track) < 1 || len(track) > 100 {
		return false
	}

	// МАКСИМАЛЬНО УПРОЩЕННАЯ валидация - принимаем почти все
	trackLower := strings.ToLower(track)

	// Исключаем ТОЛЬКО очевидные артефакты (полные совпадения)
	fullExcludes := []string{
		"music video", "youtube", "teaser poster", "official x",
		"mv release", "digital release", "physical release",
		"album release", "teaser", "poster", "n/a", "na",
		"album:", "ep:", "single:", "mini album", "mini:",
	}

	for _, exclude := range fullExcludes {
		if trackLower == exclude {
			return false
		}
	}

	// Исключаем треки, которые состоят ТОЛЬКО из служебных символов
	if regexp.MustCompile(`^[:\-–\s]+$`).MatchString(track) {
		return false
	}

	// Исключаем треки, которые начинаются с служебных слов
	prefixExcludes := []string{
		"album:", "ep:", "single:", "mini:", "teaser:", "poster:",
		"music video:", "official audio:", "digital release:",
	}

	for _, prefix := range prefixExcludes {
		if strings.HasPrefix(trackLower, prefix) {
			return false
		}
	}

	// Принимаем все остальное (включая короткие треки типа "GO!")
	return true
}

// min возвращает минимальное значение из двух
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// abs возвращает абсолютное значение
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// indexOf возвращает индекс элемента в слайсе
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

// ExtractTrackNameOldStyle использует старую простую логику для извлечения треков
func ExtractTrackNameOldStyle(eventLines []string, logger *zap.Logger) string {
	logger.Info("OldStyle: analyzing lines", zap.Int("count", len(eventLines)))

	// Ищем строки, которые начинаются с "Title Track:" (включая опечатки)
	for i, line := range eventLines {
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(lowerLine, "title track:") || strings.HasPrefix(lowerLine, "title trac:") {
			track := strings.TrimSpace(strings.TrimPrefix(line, "Title Track:"))
			track = strings.TrimSpace(strings.TrimPrefix(track, "title track:"))
			track = strings.TrimSpace(strings.TrimPrefix(track, "Title Trac:"))
			track = strings.TrimSpace(strings.TrimPrefix(track, "title trac:"))
			if track != "" && track != "n/a" {
				logger.Info("OldStyle: found Title Track", zap.String("track", track), zap.Int("line", i))
				return track
			}
		}
	}

	// Ищем треки в квадратных скобках (приоритет)
	for i, line := range eventLines {
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			start := strings.Index(line, "[")
			if start != -1 {
				end := strings.Index(line[start+1:], "]")
				if end != -1 {
					track := line[start+1 : start+1+end]
					if track != "" && track != "n/a" && isValidTrackName(track) {
						logger.Info("OldStyle: found square brackets", zap.String("track", track), zap.Int("line", i))
						return track
					}
				}
			}
		}
	}

	// Ищем названия треков в кавычках (улучшенная версия)
	for i, line := range eventLines {
		logger.Info("OldStyle: checking line", zap.String("line", line), zap.Int("index", i))

		// Сначала ищем в одинарных кавычках
		if strings.Contains(line, "'") {
			start := strings.Index(line, "'")
			if start != -1 {
				end := strings.Index(line[start+1:], "'")
				if end != -1 {
					track := line[start+1 : start+1+end]
					logger.Info("OldStyle: found single quotes", zap.String("track", track), zap.Bool("valid", isValidTrackName(track)), zap.Int("line", i))
					if track != "" && track != "n/a" && isValidTrackName(track) {
						return track
					}
				}
			}
		}

		// Затем ищем в двойных кавычках
		if strings.Contains(line, "\"") {
			start := strings.Index(line, "\"")
			if start != -1 {
				end := strings.Index(line[start+1:], "\"")
				if end != -1 {
					track := line[start+1 : start+1+end]
					logger.Info("OldStyle: found double quotes", zap.String("track", track), zap.Bool("valid", isValidTrackName(track)), zap.Int("line", i))
					if track != "" && track != "n/a" && isValidTrackName(track) {
						return track
					}
				}
			}
		}
	}

	// Ищем треки в угловых скобках
	for i, line := range eventLines {
		if strings.Contains(line, "<") && strings.Contains(line, ">") {
			start := strings.Index(line, "<")
			if start != -1 {
				end := strings.Index(line[start+1:], ">")
				if end != -1 {
					track := line[start+1 : start+1+end]
					if track != "" && track != "n/a" && isValidTrackName(track) {
						logger.Info("OldStyle: found angle brackets", zap.String("track", track), zap.Int("line", i))
						return track
					}
				}
			}
		}
	}

	logger.Info("OldStyle: no track found")
	return ""
}

// extractTrackFromDateBlock извлекает трек из блока даты (для множественных релизов)
func extractTrackFromDateBlock(block string, logger *zap.Logger) string {
	logger.Info("Extracting track from date block", zap.String("block", block))

	// Убираем лишние символы
	block = strings.TrimSpace(block)
	block = strings.TrimPrefix(block, "–")
	block = strings.TrimPrefix(block, "-")
	block = strings.TrimSpace(block)

	// ЛОГИКА КАК В СТАРОМ КОДЕ: нормализуем кавычки и ищем треки
	logger.Info("Trying extraction like old code", zap.String("block", block))

	// Нормализуем кавычки как в старом коде
	block = strings.ReplaceAll(block, "'", "'")
	block = strings.ReplaceAll(block, "'", "'")
	block = strings.ReplaceAll(block, "\u201c", "\"")
	block = strings.ReplaceAll(block, "\u201d", "\"")

	// Ищем строки с ключевыми словами как в старом коде
	lowerBlock := strings.ToLower(block)
	if strings.HasPrefix(lowerBlock, "title track:") {
		logger.Info("Found Title Track pattern, extracting track", zap.String("block", block))
		trackName := strings.TrimSpace(strings.TrimPrefix(block, "Title Track:"))
		trackName = strings.TrimSpace(strings.TrimPrefix(trackName, "title track:"))

		// Извлекаем трек из кавычек
		startDouble := strings.Index(trackName, "\"")
		endDouble := strings.LastIndex(trackName, "\"")
		if startDouble != -1 && endDouble != -1 && startDouble < endDouble {
			cleaned := trackName[startDouble+1 : endDouble]
			if cleaned != "" {
				logger.Info("Found track with Title Track pattern", zap.String("track", cleaned))
				return cleaned
			}
		}

		startSingle := strings.Index(trackName, "'")
		endSingle := strings.LastIndex(trackName, "'")
		if startSingle != -1 && endSingle != -1 && startSingle < endSingle {
			cleaned := trackName[startSingle+1 : endSingle]
			if cleaned != "" {
				logger.Info("Found track with Title Track pattern", zap.String("track", cleaned))
				return cleaned
			}
		}
	} else if strings.Contains(lowerBlock, "release") || strings.Contains(lowerBlock, "pre-release") || strings.Contains(lowerBlock, "mv release") {
		logger.Info("Found release pattern, extracting track", zap.String("block", block))

		// Извлекаем трек из двойных кавычек
		startDouble := strings.Index(block, "\"")
		endDouble := strings.LastIndex(block, "\"")
		if startDouble != -1 && endDouble != -1 && startDouble < endDouble {
			cleaned := block[startDouble+1 : endDouble]
			trackParts := strings.Fields(cleaned)
			cleaned = ""
			for _, part := range trackParts {
				if strings.ToLower(part) == "mv" || strings.ToLower(part) == "release" {
					continue
				}
				if cleaned == "" {
					cleaned = part
				} else {
					cleaned += " " + part
				}
			}
			if cleaned != "" {
				logger.Info("Found track with double quotes", zap.String("track", cleaned))
				return cleaned
			}
		}

		// Извлекаем трек из одинарных кавычек
		// Нормализуем Unicode кавычки как в старом коде
		normalizedBlock := strings.ReplaceAll(block, "\u2018", "'")
		normalizedBlock = strings.ReplaceAll(normalizedBlock, "\u2019", "'")

		startSingle := strings.Index(normalizedBlock, "'")
		endSingle := strings.LastIndex(normalizedBlock, "'")
		logger.Info("Checking single quotes", zap.Int("start", startSingle), zap.Int("end", endSingle), zap.String("normalized", normalizedBlock))
		if startSingle != -1 && endSingle != -1 && startSingle < endSingle {
			cleaned := normalizedBlock[startSingle+1 : endSingle]
			logger.Info("Extracted from single quotes", zap.String("cleaned", cleaned))
			trackParts := strings.Fields(cleaned)
			cleaned = ""
			for _, part := range trackParts {
				if strings.ToLower(part) == "mv" || strings.ToLower(part) == "release" {
					logger.Info("Skipping word", zap.String("word", part))
					continue
				}
				if cleaned == "" {
					cleaned = part
				} else {
					cleaned += " " + part
				}
			}
			logger.Info("Final cleaned track", zap.String("track", cleaned))
			if cleaned != "" {
				logger.Info("Found track with single quotes", zap.String("track", cleaned))
				return cleaned
			}
		}
	}

	// Паттерн 1: 'TRACK' MV Release (гибкий пробел)
	if match := regexp.MustCompile(`['']([^'']+)['']\s*MV\s+Release`).FindStringSubmatch(block); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) {
			logger.Info("Found track with MV Release pattern", zap.String("track", track))
			return track
		}
	}

	// Паттерн 1.1: 'TRACK' MV Release: YouTube (с двоеточием)
	if match := regexp.MustCompile(`['']([^'']+)['']\s*MV\s+Release:?\s*YouTube?`).FindStringSubmatch(block); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) {
			logger.Info("Found track with MV Release YouTube pattern", zap.String("track", track))
			return track
		}
	}

	// Паттерн 2: "TRACK" (Title Track) (гибкий пробел)
	if match := regexp.MustCompile(`[""]([^""]+)[""]\s*\(Title\s+Track\)`).FindStringSubmatch(block); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) {
			logger.Info("Found track with Title Track pattern", zap.String("track", track))
			return track
		}
	}

	// Паттерн 2.1: "TRACK" (Title Track) MV: YouTube
	if match := regexp.MustCompile(`[""]([^""]+)[""]\s*\(Title\s+Track\)\s*MV:?\s*YouTube?`).FindStringSubmatch(block); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) {
			logger.Info("Found track with Title Track MV YouTube pattern", zap.String("track", track))
			return track
		}
	}

	// Паттерн 3: "TRACK" feat. ARTIST (гибкий пробел)
	if match := regexp.MustCompile(`[""]([^""]+)[""]\s*feat\.`).FindStringSubmatch(block); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) {
			logger.Info("Found track with feat pattern", zap.String("track", track))
			return track
		}
	}

	// Паттерн 3.1: "TRACK" feat. ARTIST (полное название)
	if match := regexp.MustCompile(`[""]([^""]+)[""]\s*feat\.\s+[^"]+`).FindStringSubmatch(block); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) {
			logger.Info("Found track with full feat pattern", zap.String("track", track))
			return track
		}
	}

	// Паттерн 4: Простые кавычки (более гибкий)
	if match := regexp.MustCompile(`['']([^'']+)['']`).FindStringSubmatch(block); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) {
			logger.Info("Found track with simple quotes", zap.String("track", track))
			return track
		}
	}

	// Паттерн 5: Двойные кавычки (более гибкий)
	if match := regexp.MustCompile(`[""]([^""]+)[""]`).FindStringSubmatch(block); len(match) > 1 {
		track := strings.TrimSpace(match[1])
		if isValidTrackName(track) {
			logger.Info("Found track with double quotes", zap.String("track", track))
			return track
		}
	}

	logger.Info("No track found in date block")
	return ""
}

// isValidTrackName проверяет, является ли строка валидным названием трека
func isValidTrackName(track string) bool {
	if len(track) == 0 {
		return false
	}

	// Исключаем служебные слова (но не так строго)
	excludeWords := []string{"mv release", "album release", "teaser poster", "music video", "youtube", "twitter", "website", "official audio"}
	trackLower := strings.ToLower(track)

	for _, word := range excludeWords {
		if strings.Contains(trackLower, word) {
			return false
		}
	}

	// Трек должен содержать хотя бы одну букву или цифру
	if !regexp.MustCompile(`[a-zA-Z가-힣0-9!]`).MatchString(track) {
		return false
	}

	// Трек не должен быть слишком коротким (меньше 1 символа)
	if len(track) < 1 {
		return false
	}

	// Исключаем треки, которые состоят только из служебных символов
	if regexp.MustCompile(`^[:\-–\s]+$`).MatchString(track) {
		return false
	}

	return true
}

// extractTrackFromLine извлекает трек из строки, используя различные методы
func extractTrackFromLine(line string, logger *zap.Logger) string {
	logger.Debug("Extracting track from line", zap.String("line", line))

	// Убираем тире
	line = strings.ReplaceAll(line, "–", "")
	line = strings.ReplaceAll(line, "-", "")
	line = strings.TrimSpace(line)

	// Пробуем найти в одинарных кавычках
	singleQuoteRegex := regexp.MustCompile(`'([^']+)'`)
	matches := singleQuoteRegex.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if len(track) > 0 && !strings.Contains(strings.ToLower(track), "mv") && !strings.Contains(strings.ToLower(track), "release") {
				logger.Debug("Found track in single quotes", zap.String("track", track))
				return track
			}
		}
	}

	// Пробуем найти в двойных кавычках
	doubleQuoteRegex := regexp.MustCompile(`"([^"]+)"`)
	matches = doubleQuoteRegex.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if len(track) > 0 && !strings.Contains(strings.ToLower(track), "mv") && !strings.Contains(strings.ToLower(track), "release") {
				logger.Debug("Found track in double quotes", zap.String("track", track))
				return track
			}
		}
	}

	// Пробуем найти в угловых скобках
	angleBracketRegex := regexp.MustCompile(`<([^>]+)>`)
	matches = angleBracketRegex.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			track := strings.TrimSpace(match[1])
			if len(track) > 1 {
				logger.Debug("Found track in angle brackets", zap.String("track", track))
				return track
			}
		}
	}

	logger.Debug("No track found in line")
	return ""
}

// ParseMultipleReleases парсит строку с множественными релизами одного артиста
// Универсальная функция для всех артистов (CORTIS, JENNIE, NEWBEAT и др.)
// parsingMonth - месяц парсинга (например, "august"), релизы других месяцев игнорируются
func ParseMultipleReleases(artistName, content, parsingMonth string, logger *zap.Logger) []map[string]string {
	var releases []map[string]string

	// Декодируем HTML entities
	decoded := DecodeHTMLEntities(content)
	logger.Info("Parsing multiple releases", zap.String("artist", artistName), zap.String("parsing_month", parsingMonth), zap.String("content", decoded))

	// Преобразуем parsingMonth в номер месяца
	monthMap := map[string]string{
		"january": "01", "february": "02", "march": "03", "april": "04",
		"may": "05", "june": "06", "july": "07", "august": "08",
		"september": "09", "october": "10", "november": "11", "december": "12",
	}
	parsingMonthNum := monthMap[strings.ToLower(parsingMonth)]
	if parsingMonthNum == "" {
		logger.Warn("Unknown parsing month", zap.String("month", parsingMonth))
		return releases // Возвращаем пустой список, если месяц неизвестен
	}

	// Универсальные паттерны для поиска дат в разных форматах
	datePatterns := []string{
		// Формат: "Month XX, YYYY at XX AM/PM KST:"
		`(January|February|March|April|May|June|July|August|September|October|November|December) (\d+), (\d{4}) at (\d+) (AM|PM) KST:`,
		// Формат: "Month XX, YYYY:"
		`(January|February|March|April|May|June|July|August|September|October|November|December) (\d+), (\d{4}):`,
		// Формат: "Month XX:"
		`(January|February|March|April|May|June|July|August|September|October|November|December) (\d+):`,
	}

	var matches [][]string
	for _, pattern := range datePatterns {
		re := regexp.MustCompile(pattern)
		found := re.FindAllStringSubmatch(decoded, -1)
		matches = append(matches, found...)
	}

	logger.Info("Found date matches", zap.Int("count", len(matches)))

	for i, match := range matches {
		if len(match) >= 3 {
			month := match[1] // January, February, etc.
			day := match[2]   // 1, 2, etc.

			// Определяем год в зависимости от формата
			var year string
			if len(match) >= 4 && match[3] != "" {
				year = match[3] // Любой год из данных
			} else {
				// Если год не указан, используем текущий год
				year = time.Now().Format("2006")
			}

			// Преобразуем месяц в номер
			monthMap := map[string]string{
				"January": "01", "February": "02", "March": "03", "April": "04",
				"May": "05", "June": "06", "July": "07", "August": "08",
				"September": "09", "October": "10", "November": "11", "December": "12",
			}
			monthNum := monthMap[month]
			if monthNum == "" {
				logger.Warn("Unknown month", zap.String("month", month))
				continue
			}

			// КРИТИЧЕСКАЯ ПРОВЕРКА: Пропускаем релизы с датами других месяцев
			if monthNum != parsingMonthNum {
				logger.Debug("Skipping release from different month",
					zap.String("artist", artistName),
					zap.String("release_month", monthNum),
					zap.String("parsing_month", parsingMonthNum),
					zap.String("date", month+" "+day+", "+year))
				continue
			}

			// Форматируем дату в DD.MM.YY
			if len(day) == 1 {
				day = "0" + day
			}
			// Берем последние 2 цифры года
			yearShort := year[2:]
			date := day + "." + monthNum + "." + yearShort

			// Ищем трек после этой даты
			dateStr := match[0]
			dateIndex := strings.Index(decoded, dateStr)
			if dateIndex != -1 {
				// Берем текст после даты до следующей даты или конца строки
				afterDate := decoded[dateIndex+len(dateStr):]
				// Ищем следующую дату (все месяцы)
				nextDateIndex := -1
				allMonths := []string{"January", "February", "March", "April", "May", "June",
					"July", "August", "September", "October", "November", "December"}
				for _, monthName := range allMonths {
					if idx := strings.Index(afterDate, monthName); idx != -1 {
						if nextDateIndex == -1 || idx < nextDateIndex {
							nextDateIndex = idx
						}
					}
				}
				if nextDateIndex != -1 {
					afterDate = afterDate[:nextDateIndex]
				}

				// Убираем лишние пробелы и переносы строк
				afterDate = strings.TrimSpace(afterDate)

				// Извлекаем трек из этого фрагмента
				track := ExtractTrackFromQuotes(afterDate, logger)

				// Если основной парсер не нашел трек, пробуем Fallback Parser
				if track == "" {
					logger.Info("Primary parser failed, trying fallback parser", zap.String("artist", artistName), zap.Int("index", i))
					track = FallbackTrackExtraction(afterDate, logger)
				}

				if track != "" {
					// Извлекаем YouTube ссылку для этого конкретного релиза из текста
					youtubeLink := extractYouTubeLinkFromText(afterDate, logger)

					release := map[string]string{
						"artist":  artistName,
						"date":    date,
						"track":   track,
						"youtube": youtubeLink,
					}
					releases = append(releases, release)
					logger.Info("Found multiple release", zap.Int("index", i), zap.String("artist", artistName), zap.String("date", date), zap.String("track", track), zap.String("youtube", youtubeLink))
					logger.Info("DEBUG: Added release to array", zap.Int("total_releases", len(releases)), zap.String("last_release_date", date), zap.String("last_release_track", track), zap.String("youtube", youtubeLink))
				}
			}
		}
	}

	logger.Info("DEBUG: Returning releases", zap.Int("count", len(releases)))
	for i, release := range releases {
		logger.Info("DEBUG: Release in array", zap.Int("index", i), zap.String("date", release["date"]), zap.String("track", release["track"]))
	}
	return releases
}

// GetReleaseConfig возвращает конфигурацию релизов
func GetReleaseConfig() *model.ReleaseConfig {
	return model.NewReleaseConfig()
}

// FormatTimeKST форматирует время KST
func FormatTimeKST(timeText string, logger *zap.Logger) (string, error) {
	config := GetReleaseConfig()
	utils := model.NewReleaseUtils()

	parsedTime, err := utils.ParseReleaseTime(timeText)
	if err != nil {
		logger.Debug("Failed to parse KST time", zap.String("time", timeText), zap.Error(err))
		return "", err
	}

	return parsedTime.Format(config.TimeFormat()), nil
}

// ConvertKSTtoMSK конвертирует время из KST в MSK
func ConvertKSTtoMSK(kstTime string, logger *zap.Logger) (string, error) {
	if kstTime == "" {
		return "", nil
	}

	utils := model.NewReleaseUtils()
	return utils.ConvertKSTToMSKString(kstTime)
}

// FormatDate форматирует дату
func FormatDate(dateText string, logger *zap.Logger) (string, error) {
	utils := model.NewReleaseUtils()

	parsedDate, err := utils.ParseReleaseDate(dateText)
	if err != nil {
		logger.Debug("Failed to parse date", zap.String("date", dateText), zap.Error(err))
		return "", err
	}

	return utils.FormatReleaseDate(parsedDate), nil
}

// FormatDateWithYear форматирует дату с указанным годом
func FormatDateWithYear(dateText string, year string, logger *zap.Logger) (string, error) {
	utils := model.NewReleaseUtils()

	parsedDate, err := utils.ParseReleaseDateWithYear(dateText, year)
	if err != nil {
		logger.Debug("Failed to parse date with year", zap.String("date", dateText), zap.String("year", year), zap.Error(err))
		return "", err
	}

	return utils.FormatReleaseDate(parsedDate), nil
}

// ExtractAlbumName извлекает название альбома из строк событий
func ExtractAlbumName(eventLines []string, startIndex, endIndex int, logger *zap.Logger) string {
	for i := startIndex; i < endIndex && i < len(eventLines); i++ {
		line := strings.ToLower(eventLines[i])
		if strings.HasPrefix(line, "album:") {
			album := strings.TrimSpace(strings.TrimPrefix(eventLines[i], "Album:"))
			album = strings.TrimSpace(strings.TrimPrefix(album, "album:"))
			if album != "" && album != "n/a" {
				return album
			}
		}
		// Проверяем, содержит ли строка информацию об альбоме после двоеточия
		if strings.Contains(line, ":") {
			parts := strings.SplitN(eventLines[i], ":", 2)
			if len(parts) == 2 {
				content := strings.TrimSpace(parts[1])
				// Если содержимое содержит слова, указывающие на альбом
				if strings.Contains(strings.ToLower(content), "album") ||
					strings.Contains(strings.ToLower(content), "ep") ||
					strings.Contains(strings.ToLower(content), "single") ||
					strings.Contains(strings.ToLower(content), "mini") {
					return content
				}
			}
		}
	}
	return "N/A"
}

// ExtractAlbumNameFromAllLines извлекает название альбома из всех строк артиста
func ExtractAlbumNameFromAllLines(allLines []string, logger *zap.Logger) string {
	// Сначала ищем строки, которые начинаются с "Album:"
	for _, line := range allLines {
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(lowerLine, "album:") {
			album := strings.TrimSpace(strings.TrimPrefix(line, "Album:"))
			album = strings.TrimSpace(strings.TrimPrefix(album, "album:"))
			if album != "" && album != "n/a" {
				return album
			}
		}
	}

	// Если не нашли "Album:", ищем строки с информацией об альбоме
	for _, line := range allLines {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				content := strings.TrimSpace(parts[1])
				// Если содержимое содержит слова, указывающие на альбом
				if strings.Contains(strings.ToLower(content), "album") ||
					strings.Contains(strings.ToLower(content), "ep") ||
					strings.Contains(strings.ToLower(content), "single") ||
					strings.Contains(strings.ToLower(content), "mini") {
					return content
				}
			}
		}
	}

	return "N/A"
}

// ExtractTrackName извлекает название трека из строк событий
// НОВАЯ ЛОГИКА: Используем подход ExtractTrackNameFromAllLines как основной
func ExtractTrackName(eventLines []string, startIndex, endIndex int, logger *zap.Logger) string {
	// Сначала пробуем старую логику для обратной совместимости
	for i := startIndex; i < endIndex && i < len(eventLines); i++ {
		line := strings.ToLower(eventLines[i])
		if strings.HasPrefix(line, "title track:") {
			track := strings.TrimSpace(strings.TrimPrefix(eventLines[i], "Title Track:"))
			track = strings.TrimSpace(strings.TrimPrefix(track, "title track:"))
			track = cleanTrackName(track) // КРИТИЧНО: Очищаем артефакты
			if track != "" && track != "n/a" {
				return track
			}
		}
	}

	// НОВАЯ ЛОГИКА: Используем ExtractTrackNameFromAllLines для всех строк
	// Это работает лучше, потому что анализирует весь контент артиста
	return ExtractTrackNameFromAllLines(eventLines, logger)
}

// ExtractTrackNameFromAllLines извлекает название трека из всех строк артиста
// УЛУЧШЕННАЯ ВЕРСИЯ: Объединяем лучшие паттерны из унифицированного экстрактора
func ExtractTrackNameFromAllLines(allLines []string, logger *zap.Logger) string {
	content := strings.Join(allLines, "\n")
	logger.Info("ExtractTrackNameFromAllLines: analyzing content", zap.String("content", content[:min(100, len(content))]))

	// 1. ПРИОРИТЕТ: Title Track паттерны
	for _, line := range allLines {
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(lowerLine, "title track:") {
			track := strings.TrimSpace(strings.TrimPrefix(line, "Title Track:"))
			track = strings.TrimSpace(strings.TrimPrefix(track, "title track:"))
			track = cleanTrackName(track)
			if track != "" && track != "n/a" && isValidUnifiedTrack(track) {
				logger.Info("ExtractTrackNameFromAllLines: found Title Track", zap.String("track", track))
				return track
			}
		}
	}

	// 2. УНИФИЦИРОВАННЫЙ ЭКСТРАКТОР: Используем лучшие паттерны
	unifiedTrack := extractTrackUnified(content, logger)
	if unifiedTrack != "" {
		logger.Info("ExtractTrackNameFromAllLines: found with unified extractor", zap.String("track", unifiedTrack))
		return unifiedTrack
	}

	// 3. ExtractTrackFromQuotes: Для каждого трека отдельно
	for _, line := range allLines {
		track := ExtractTrackFromQuotes(line, logger)
		if track != "" && isValidUnifiedTrack(track) {
			logger.Info("ExtractTrackNameFromAllLines: found with ExtractTrackFromQuotes", zap.String("track", track))
			return track
		}
	}

	// 3.5. ДОПОЛНИТЕЛЬНАЯ ПРОВЕРКА: Если ExtractTrackFromQuotes не сработал, пробуем напрямую
	for _, line := range allLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Прямой поиск треков в кавычках
		if match := regexp.MustCompile(`['']([^'']+)['']`).FindStringSubmatch(line); len(match) > 1 {
			track := cleanTrackName(match[1])
			if track != "" && isValidUnifiedTrack(track) {
				logger.Info("ExtractTrackNameFromAllLines: found with direct single quotes", zap.String("track", track))
				return track
			}
		}

		if match := regexp.MustCompile(`[""]([^""]+)[""]`).FindStringSubmatch(line); len(match) > 1 {
			track := cleanTrackName(match[1])
			if track != "" && isValidUnifiedTrack(track) {
				logger.Info("ExtractTrackNameFromAllLines: found with direct double quotes", zap.String("track", track))
				return track
			}
		}
	}

	// 3.5. ДОПОЛНИТЕЛЬНАЯ ЛОГИКА: Простые паттерны для сложных случаев
	for _, line := range allLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Паттерн: September X: "Track" или September X: 'Track'
		if match := regexp.MustCompile(`(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d+:\s*["']([^"']+)["']`).FindStringSubmatch(line); len(match) > 1 {
			track := cleanTrackName(match[1])
			if track != "" && isValidUnifiedTrack(track) {
				logger.Info("ExtractTrackNameFromAllLines: found with date pattern", zap.String("track", track))
				return track
			}
		}

		// Паттерн: Digital Release без кавычек
		if strings.Contains(strings.ToLower(line), "digital release") {
			// Попробуем найти название трека в этой строке или соседних
			for j, nearLine := range allLines {
				if nearLine == line {
					continue
				}
				if abs(j-indexOf(allLines, line)) <= 2 { // В пределах 2 строк
					nearTrack := ExtractTrackFromQuotes(nearLine, logger)
					if nearTrack != "" && isValidUnifiedTrack(nearTrack) {
						logger.Info("ExtractTrackNameFromAllLines: found near Digital Release", zap.String("track", nearTrack))
						return nearTrack
					}
				}
			}
		}

		// НОВЫЙ ПАТТЕРН: Album Physical Release - ищем трек в соседних строках
		if strings.Contains(strings.ToLower(line), "album physical release") {
			// Ищем трек в соседних строках
			for j, nearLine := range allLines {
				if nearLine == line {
					continue
				}
				if abs(j-indexOf(allLines, line)) <= 3 { // В пределах 3 строк
					nearTrack := ExtractTrackFromQuotes(nearLine, logger)
					if nearTrack != "" && isValidUnifiedTrack(nearTrack) {
						logger.Info("ExtractTrackNameFromAllLines: found near Album Physical Release", zap.String("track", nearTrack))
						return nearTrack
					}
				}
			}
		}

		// НОВЫЙ ПАТТЕРН: Pre-release - ищем трек в той же строке
		if strings.Contains(strings.ToLower(line), "pre-release") {
			// Ищем трек в той же строке
			track := ExtractTrackFromQuotes(line, logger)
			if track != "" && isValidUnifiedTrack(track) {
				logger.Info("ExtractTrackNameFromAllLines: found in Pre-release line", zap.String("track", track))
				return track
			}
		}

		// НОВЫЙ ПАТТЕРН: Японские символы 『』 - извлекаем название альбома
		if strings.Contains(line, "『") && strings.Contains(line, "』") {
			if match := regexp.MustCompile(`『([^』]+)』`).FindStringSubmatch(line); len(match) > 1 {
				track := cleanTrackName(match[1])
				if track != "" && isValidUnifiedTrack(track) {
					logger.Info("ExtractTrackNameFromAllLines: found Japanese brackets", zap.String("track", track))
					return track
				}
			}
		}
	}

	// 4. Fallback Parser: Последний шанс
	fallbackTrack := FallbackTrackExtraction(content, logger)
	if fallbackTrack != "" {
		logger.Info("ExtractTrackNameFromAllLines: found with fallback", zap.String("track", fallbackTrack))
		return fallbackTrack
	}

	logger.Info("ExtractTrackNameFromAllLines: no track found")
	return "N/A"
}

// ExtractYouTubeLinkFromEvent извлекает YouTube ссылку из события
func ExtractYouTubeLinkFromEvent(e *colly.HTMLElement, startIndex, endIndex int, logger *zap.Logger) string {
	// Ищем ссылки на YouTube в указанном диапазоне
	var youtubeLink string

	e.ForEach("a[href]", func(_ int, link *colly.HTMLElement) {
		href := link.Attr("href")
		if strings.Contains(href, "youtube.com") || strings.Contains(href, "youtu.be") {
			youtubeLink = href
		}
	})

	if youtubeLink != "" {
		return youtubeLink
	}

	return ""
}

// extractYouTubeLinkFromText извлекает YouTube ссылку из HTML текста
func extractYouTubeLinkFromText(text string, logger *zap.Logger) string {
	// Ищем YouTube ссылки в HTML атрибутах href
	youtubeRegex := regexp.MustCompile(`href="(https?://(?:www\.)?(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]+(?:\?[^"]*)?))"`)
	matches := youtubeRegex.FindStringSubmatch(text)

	if len(matches) > 1 {
		// Возвращаем найденную ссылку
		link := matches[1]
		logger.Info("Found YouTube link in HTML", zap.String("link", link), zap.String("text", text))
		return link
	}

	// Fallback: ищем YouTube ссылки в обычном тексте
	youtubeRegexText := regexp.MustCompile(`https?://(?:www\.)?(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]+(?:\?[^"]*)?)`)
	matchesText := youtubeRegexText.FindStringSubmatch(text)

	if len(matchesText) > 0 {
		link := matchesText[0]
		logger.Info("Found YouTube link in text", zap.String("link", link), zap.String("text", text))
		return link
	}

	logger.Info("No YouTube link found", zap.String("text", text))
	return ""
}
