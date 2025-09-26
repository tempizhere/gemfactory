package scraper

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
	"time"

	"gemfactory/internal/model"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// ParsedRelease представляет результат парсинга одного релиза
type ParsedRelease struct {
	Artist     string `json:"artist"`
	Date       string `json:"date"`
	Track      string `json:"track"`
	Album      string `json:"album"`
	YouTubeURL string `json:"youtube"`
}

// ParseResult представляет результат парсинга блока
type ParseResult struct {
	Releases []ParsedRelease `json:"releases"`
	Success  bool            `json:"success"`
	Error    string          `json:"error,omitempty"`
}

// cleanYouTubeURL очищает YouTube URL от лишних параметров
func cleanYouTubeURL(url string) string {
	// Удаляем параметры типа ?si=... и другие tracking параметры
	url = regexp.MustCompile(`\?si=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`&si=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`\?t=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`&t=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`\?list=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`\?v=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`&v=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`\?feature=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`&feature=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`\?index=[^&]*`).ReplaceAllString(url, "")
	url = regexp.MustCompile(`&index=[^&]*`).ReplaceAllString(url, "")

	// Удаляем оставшиеся пустые параметры
	url = regexp.MustCompile(`\?&`).ReplaceAllString(url, "?")
	url = regexp.MustCompile(`\?$`).ReplaceAllString(url, "")

	return url
}

// cleanHTMLBlock очищает HTML блок - извлекает дату, артиста и релизы в формате <event>
// cleanHTMLBlock очищает HTML блок и преобразует его в структурированный формат
// Входной формат: <tr><td>дата</td><td>артист + релизы</td></tr>
// Выходной формат: <event><date>дата</date><artist>артист</artist><need_unparse>релизы</need_unparse></event>
func cleanHTMLBlock(htmlStr string) string {
	// 1. Извлекаем дату из первого mark тега (левая колонка)
	dateRe := regexp.MustCompile(`<mark[^>]*>([^<]+)</mark>`)
	dateMatch := dateRe.FindStringSubmatch(htmlStr)
	date := ""
	if len(dateMatch) > 1 {
		date = strings.TrimSpace(dateMatch[1])
	}

	// 2. Извлекаем имя артиста с приоритетом mark тега
	// Приоритет 1: содержимое mark тега внутри strong (например, <strong><mark>Rosanna</mark></strong>)
	markRe := regexp.MustCompile(`<strong><mark[^>]*>([^<]+)</mark></strong>`)
	markMatch := markRe.FindStringSubmatch(htmlStr)
	artist := ""
	if len(markMatch) > 1 {
		artist = markMatch[1]
	} else {
		// Приоритет 2: содержимое strong тега (для случаев без mark тега)
		strongRe := regexp.MustCompile(`<strong>([^<]+)</strong>`)
		strongMatch := strongRe.FindStringSubmatch(htmlStr)
		if len(strongMatch) > 1 {
			artist = strongMatch[1]
		}
	}

	// Обрабатываем извлеченное имя артиста
	if artist != "" {
		// Декодируем HTML-сущности: &amp;TEAM -> &TEAM, &lt; -> <, и т.д.
		artist = html.UnescapeString(artist)
		// Сохраняем все символы: скобки, специальные символы, эмодзи
		// Примеры: ALL(H)OURS, tripleS ∞!, IRENE & SEULGI
		artist = strings.TrimSpace(artist)
	}

	// 3. Извлекаем и очищаем информацию о релизах из правой колонки
	releasesRe := regexp.MustCompile(`<td class="has-text-align-left"[^>]*>(.*?)</td>`)
	releasesMatch := releasesRe.FindStringSubmatch(htmlStr)
	releases := ""
	if len(releasesMatch) > 1 {
		releases = releasesMatch[1]

		// Удаляем служебную информацию (Teaser Poster и всё после него)
		releases = regexp.MustCompile(`(?is)Teaser Poster:.*`).ReplaceAllString(releases, "")

		// Декодируем HTML-сущности: &amp; -> &, &lt; -> <, &quot; -> ", и т.д.
		releases = html.UnescapeString(releases)

		// Преобразуем HTML переносы строк в обычные переносы
		releases = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(releases, "\n")

		// Удаляем форматирующие теги (strong, mark, span) - они не нужны для парсинга
		releases = regexp.MustCompile(`</?(strong|mark|span)[^>]*>`).ReplaceAllString(releases, "")

		// Умная очистка HTML тегов:
		// - Сохраняем <a> ссылки
		// - Сохраняем теги без атрибутов (могут быть частью названий: <unevermet>, <Club Icarus Remix>)
		// - Удаляем остальные теги
		releases = regexp.MustCompile(`</?[^>]+>`).ReplaceAllStringFunc(releases, func(tag string) string {
			// Сохраняем ссылки
			if strings.HasPrefix(tag, "<a ") || strings.HasPrefix(tag, "</a>") {
				return tag
			}
			// Сохраняем теги без атрибутов (могут быть частью названий альбомов/треков)
			if strings.HasPrefix(tag, "<") && !strings.Contains(tag, "=") {
				return tag
			}
			// Удаляем все остальные теги
			return ""
		})

		// Очищаем YouTube ссылки от tracking параметров (si=, t=, и т.д.)
		releases = regexp.MustCompile(`<a href="([^"]*(?:youtu\.be|youtube\.com)[^"]*)"[^>]*>`).ReplaceAllStringFunc(releases, func(match string) string {
			hrefRegex := regexp.MustCompile(`href="([^"]*)"`)
			hrefMatch := hrefRegex.FindStringSubmatch(match)
			if len(hrefMatch) > 1 {
				cleanedURL := cleanYouTubeURL(hrefMatch[1])
				return strings.Replace(match, hrefMatch[1], cleanedURL, 1)
			}
			return match
		})

		// Удаляем лишние пробелы в начале и конце
		releases = strings.TrimSpace(releases)
	}

	// 4. Формируем структурированный результат в формате <event>, используется для дальнейшего парсинга (умный парсинг или LLM)
	result := fmt.Sprintf(
		"<event>\n<date>%s</date>\n<artist>%s</artist>\n<need_unparse>\n%s\n</need_unparse>\n</event>",
		date, artist, releases,
	)

	return result
}

// isSimpleCase проверяет, является ли блок простым случаем для локального парсинга
func IsSimpleCase(htmlStr string, logger *zap.Logger) bool {
	// 1. Проверяем количество дат в блоке
	dateCount := countDatesInBlock(htmlStr)
	logger.Debug("Date count check", zap.Int("count", dateCount))
	if dateCount > 1 {
		logger.Debug("Multiple dates detected", zap.Int("count", dateCount))
		return false // Множественные даты - сложный случай
	}

	// 2. Проверяем три простых случая:
	// - Title Track + Album
	// - Title Track + OST
	// - Album, без Title Track и без YouTube ссылок

	hasTitleTrack := regexp.MustCompile(`(?i)title track:\s*[^\n]+`).MatchString(htmlStr)
	hasAlbum := regexp.MustCompile(`(?i)album:\s*[^\n]+`).MatchString(htmlStr)
	hasOST := regexp.MustCompile(`(?i)ost:\s*[^\n]+`).MatchString(htmlStr)
	hasYouTube := regexp.MustCompile(`https://(?:youtu\.be/|youtube\.com/)[^\s"']+`).MatchString(htmlStr)

	logger.Debug("Simple case checks",
		zap.Bool("has_title_track", hasTitleTrack),
		zap.Bool("has_album", hasAlbum),
		zap.Bool("has_ost", hasOST),
		zap.Bool("has_youtube", hasYouTube))

	// Случай 1: Title Track + Album
	if hasTitleTrack && hasAlbum {
		logger.Debug("Simple case: Title Track + Album")
		return true
	}

	// Случай 2: Title Track + OST
	if hasTitleTrack && hasOST {
		logger.Debug("Simple case: Title Track + OST")
		return true
	}

	// Случай 3: Album, без Title Track и без YouTube ссылок
	if hasAlbum && !hasTitleTrack && !hasYouTube {
		logger.Debug("Simple case: Album only, no Title Track, no YouTube")
		return true
	}

	logger.Debug("Complex case detected")
	return false // Сложный случай
}

// countDatesInBlock подсчитывает количество дат в блоке
func countDatesInBlock(htmlStr string) int {
	// 1. Ищем даты в <date> тегах (для очищенных блоков)
	dateTagRegex := regexp.MustCompile(`<date>([^<]+)</date>`)
	dateTagMatches := dateTagRegex.FindAllString(htmlStr, -1)
	if len(dateTagMatches) > 0 {
		// Также считаем даты в <need_unparse> теге, но исключаем справочные даты
		needUnparseRegex := regexp.MustCompile(`<need_unparse>([\s\S]*?)</need_unparse>`)
		needUnparseMatches := needUnparseRegex.FindAllStringSubmatch(htmlStr, -1)

		totalDates := len(dateTagMatches)
		for _, match := range needUnparseMatches {
			if len(match) > 1 {
				content := match[1]
				// Ищем даты в формате "Month Day, Year", но исключаем справочные даты
				datePattern := `\b(january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2},\s+\d{4}\b`
				re := regexp.MustCompile(datePattern)
				allDates := re.FindAllString(strings.ToLower(content), -1)

				// Исключаем даты в контексте "Album Release:", "Digital Release:", "CD Release:" и т.д.
				referenceDatePattern := `(?i)(album release|digital release|cd release|mv release|pre-release|ost release):\s*[^:]*\b(january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2},\s+\d{4}\b`
				referenceRegex := regexp.MustCompile(referenceDatePattern)
				referenceMatches := referenceRegex.FindAllString(strings.ToLower(content), -1)

				// Подсчитываем только даты, которые НЕ являются справочными
				referenceDates := make(map[string]bool)
				for _, refMatch := range referenceMatches {
					// Извлекаем дату из справочного контекста
					dateInRef := re.FindString(refMatch)
					if dateInRef != "" {
						referenceDates[dateInRef] = true
					}
				}

				// Считаем только даты, которые не являются справочными
				for _, date := range allDates {
					if !referenceDates[date] {
						totalDates++
					}
				}
			}
		}
		return totalDates
	}

	// 2. Ищем даты в формате "Month Day, Year" или "YYYY.MM.DD" (для оригинальных блоков)
	datePatterns := []string{
		`\b(january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2},\s+\d{4}\b`,
		`\b\d{4}\.\d{2}\.\d{2}\b`,
	}

	count := 0
	lowerHTML := strings.ToLower(htmlStr)

	for _, pattern := range datePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(lowerHTML, -1)
		count += len(matches)
	}

	return count
}

// extractSimpleRelease извлекает данные для простого случая
func ExtractSimpleRelease(htmlStr, month, year string, logger *zap.Logger) (*ParseResult, error) {
	// 1. Извлекаем дату
	date := extractDate(htmlStr, month, year, logger)
	logger.Info("Extracted date", zap.String("date", date))
	if date == "" {
		logger.Info("No date found, returning failure")
		return &ParseResult{
			Releases: []ParsedRelease{},
			Success:  false,
			Error:    "No date found",
		}, nil
	}

	// 2. Извлекаем артиста
	artist := extractArtist(htmlStr, logger)
	logger.Info("Extracted artist", zap.String("artist", artist))
	if artist == "" {
		logger.Info("No artist found, returning failure")
		return &ParseResult{
			Releases: []ParsedRelease{},
			Success:  false,
			Error:    "No artist found",
		}, nil
	}

	// 3. Извлекаем трек (может быть пустым для случаев с только альбомом)
	track := extractTrack(htmlStr, logger)
	logger.Info("Extracted track", zap.String("track", track))

	// 4. Извлекаем альбом
	album := extractAlbum(htmlStr, logger)
	logger.Info("Extracted album", zap.String("album", album))

	// 5. Извлекаем YouTube ссылку
	youtube := extractYouTubeLink(htmlStr, logger)
	logger.Info("Extracted youtube", zap.String("youtube", youtube))

	logger.Debug("Extracted simple release",
		zap.String("artist", artist),
		zap.String("date", date),
		zap.String("track", track),
		zap.String("album", album),
		zap.String("youtube", youtube))

	return &ParseResult{
		Releases: []ParsedRelease{{
			Artist:     artist,
			Date:       date,
			Track:      track,
			Album:      album,
			YouTubeURL: youtube,
		}},
		Success: true,
	}, nil
}

// extractDate извлекает дату из HTML блока
func extractDate(htmlStr, month, year string, logger *zap.Logger) string {
	// 1. Ищем дату в <date> теге (для очищенных блоков)
	dateRegex := regexp.MustCompile(`<date>([^<]+)</date>`)
	matches := dateRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 1 {
		dateStr := strings.TrimSpace(matches[1])
		logger.Debug("Found date in date tag", zap.String("date", dateStr))

		// Парсим дату в формате "Month Day, Year" и конвертируем в DD.MM.YY
		parsedDate, err := parseEnglishDate(dateStr, year)
		if err != nil {
			logger.Error("Failed to parse date", zap.String("date", dateStr), zap.Error(err))
			return ""
		}
		return parsedDate
	}

	// 2. Ищем дату в <mark> теге (для оригинальных HTML блоков)
	markRegex := regexp.MustCompile(`<mark[^>]*>([^<]+)</mark>`)
	matches = markRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 1 {
		dateStr := strings.TrimSpace(matches[1])
		logger.Debug("Found date in mark tag", zap.String("date", dateStr))

		// Парсим дату в формате "Month Day, Year" и конвертируем в DD.MM.YY
		parsedDate, err := parseEnglishDate(dateStr, year)
		if err != nil {
			logger.Error("Failed to parse date", zap.String("date", dateStr), zap.Error(err))
			return ""
		}
		return parsedDate
	}

	return ""
}

// parseEnglishDate парсит дату в формате "Month Day, Year" и возвращает DD.MM.YY
func parseEnglishDate(dateStr, year string) (string, error) {
	// Маппинг месяцев
	monthMap := map[string]string{
		"january": "01", "february": "02", "march": "03", "april": "04",
		"may": "05", "june": "06", "july": "07", "august": "08",
		"september": "09", "october": "10", "november": "11", "december": "12",
	}

	// Парсим дату в формате "Month Day, Year"
	re := regexp.MustCompile(`(?i)(\w+)\s+(\d{1,2}),\s+(\d{4})`)
	matches := re.FindStringSubmatch(dateStr)
	if len(matches) != 4 {
		return "", fmt.Errorf("invalid date format: %s", dateStr)
	}

	monthName := strings.ToLower(matches[1])
	day := matches[2]
	dateYear := matches[3]

	monthNum, exists := monthMap[monthName]
	if !exists {
		return "", fmt.Errorf("unknown month: %s", monthName)
	}

	// Форматируем день с ведущим нулем
	if len(day) == 1 {
		day = "0" + day
	}

	// Берем последние 2 цифры года
	yearShort := dateYear[len(dateYear)-2:]

	return fmt.Sprintf("%s.%s.%s", day, monthNum, yearShort), nil
}

// extractArtist извлекает артиста из HTML блока
func extractArtist(htmlStr string, logger *zap.Logger) string {
	// Ищем артиста в <artist> теге (после очистки HTML)
	artistRegex := regexp.MustCompile(`<artist>([^<]+)</artist>`)
	matches := artistRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 1 {
		artist := strings.TrimSpace(matches[1])
		logger.Debug("Found artist in artist tag", zap.String("artist", artist))
		return artist
	}

	// Fallback: ищем в <strong><mark> теге (для неочищенного HTML)
	artistRegex = regexp.MustCompile(`<strong><mark[^>]*>([^<]+)</mark></strong>`)
	matches = artistRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 1 {
		artist := strings.TrimSpace(matches[1])
		logger.Debug("Found artist in strong/mark tag", zap.String("artist", artist))
		return artist
	}

	// Fallback: ищем в <strong> теге
	strongRegex := regexp.MustCompile(`<strong>([^<]+)</strong>`)
	matches = strongRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 1 {
		artist := strings.TrimSpace(matches[1])
		logger.Debug("Found artist in strong tag", zap.String("artist", artist))
		return artist
	}

	return ""
}

// extractTrack извлекает трек из HTML блока
func extractTrack(htmlStr string, logger *zap.Logger) string {
	// 1. Ищем "Title Track:" - берем все до следующей строки или конца
	titleTrackRegex := regexp.MustCompile(`(?i)title track:\s*([^\n]+)`)
	matches := titleTrackRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 1 {
		track := strings.TrimSpace(matches[1])
		// НЕ убираем HTML теги - они могут быть частью названия (например, <unevermet>)
		track = cleanTrackName(track)
		logger.Debug("Found track from Title Track", zap.String("track", track))
		return track
	}

	// 2. Ищем "Album Release" или "MV Release"
	releaseRegex := regexp.MustCompile(`(?i)(album|mv)\s+release`)
	if releaseRegex.MatchString(htmlStr) {
		track := "Album & MV Release"
		logger.Debug("Found general release", zap.String("track", track))
		return track
	}

	// 3. Если есть только "Album:" без "Title Track:", возвращаем пустую строку
	albumRegex := regexp.MustCompile(`(?i)album:\s*[^\n]+`)
	if albumRegex.MatchString(htmlStr) {
		logger.Debug("Found album without title track, returning empty track")
		return ""
	}

	return ""
}

// extractAlbum извлекает альбом из HTML блока
func extractAlbum(htmlStr string, logger *zap.Logger) string {
	// 1. Ищем "Album:"
	albumRegex := regexp.MustCompile(`(?i)album:\s*([^\n]+)`)
	matches := albumRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 1 {
		album := strings.TrimSpace(matches[1])
		logger.Debug("Found album", zap.String("album", album))
		return album
	}

	// 2. Ищем "OST:"
	ostRegex := regexp.MustCompile(`(?i)ost:\s*([^\n]+)`)
	matches = ostRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 1 {
		album := strings.TrimSpace(matches[1])
		logger.Debug("Found OST", zap.String("album", album))
		return album
	}

	return ""
}

// extractYouTubeLink извлекает YouTube ссылку из HTML блока
func extractYouTubeLink(htmlStr string, logger *zap.Logger) string {
	// Ищем YouTube ссылки
	youtubeRegex := regexp.MustCompile(`https://(?:youtu\.be/|youtube\.com/)[^\s"']+`)
	matches := youtubeRegex.FindStringSubmatch(htmlStr)
	if len(matches) > 0 {
		url := matches[0]
		// Исключаем каналы
		if !strings.Contains(url, "/@") {
			cleanedURL := cleanYouTubeURL(url)
			logger.Debug("Found YouTube link", zap.String("url", cleanedURL))
			return cleanedURL
		}
	}

	return ""
}

// cleanTrackName очищает название трека
func cleanTrackName(track string) string {
	// Убираем кавычки
	track = strings.Trim(track, `"'`)

	// Убираем "MV" и "Release" из названия
	words := strings.Fields(track)
	var cleaned []string
	for _, word := range words {
		lowerWord := strings.ToLower(word)
		if lowerWord != "mv" && lowerWord != "release" {
			cleaned = append(cleaned, word)
		}
	}
	return strings.Join(cleaned, " ")
}

// TestExtractTrack тестирует извлечение трека (для отладки)
func TestExtractTrack(htmlStr string) string {
	// Создаем временный логгер для тестирования
	logger := zap.NewNop()
	return extractTrack(htmlStr, logger)
}

// TestIsSimpleCase тестирует определение простого случая (для отладки)
func TestIsSimpleCase(htmlStr string) bool {
	// Создаем временный логгер для тестирования
	logger := zap.NewNop()
	return IsSimpleCase(htmlStr, logger)
}

// TestExtractDate тестирует извлечение даты (для отладки)
func TestExtractDate(htmlStr, month, year string) string {
	// Создаем временный логгер для тестирования
	logger := zap.NewNop()
	return extractDate(htmlStr, month, year, logger)
}

// llmParseBlocks отправляет блоки в LLM для парсинга
func (f *fetcherImpl) llmParseBlocks(ctx context.Context, blocks []string, month, year string) ([]ParsedRelease, error) {
	if len(blocks) == 0 {
		return []ParsedRelease{}, nil
	}

	// Объединяем все блоки через точку с запятой
	htmlBlocks := strings.Join(blocks, "; ")

	// Отправляем в LLM
	llmClient := f.llmClient
	if llmClient == nil {
		return nil, fmt.Errorf("LLM client not available")
	}

	response, err := llmClient.ParseMultiRelease(ctx, htmlBlocks, month)
	if err != nil {
		return nil, fmt.Errorf("failed to parse with LLM: %w", err)
	}

	// Конвертируем в ParsedRelease
	var releases []ParsedRelease
	for _, release := range response.Releases {
		releases = append(releases, ParsedRelease{
			Artist:     release.Artist,
			Date:       release.Date,
			Track:      release.Track,
			Album:      release.Album,
			YouTubeURL: release.YouTubeURL,
		})
	}

	return releases, nil
}

// ParseMonthlyPage parses a monthly schedule page (новая LLM-основанная логика)
func (f *fetcherImpl) ParseMonthlyPage(ctx context.Context, url, month, year string, artists map[string]bool) ([]Release, error) {
	monthNum, ok := f.getMonthNumber(strings.ToLower(month))
	if !ok {
		f.logger.Error("Unknown month", zap.String("month", month))
		return nil, fmt.Errorf("unknown month: %s", month)
	}

	// Собираем все блоки с артистами для LLM обработки
	var artistBlocks []ArtistBlock
	var mu sync.Mutex
	rowCount := 0
	var contextCancelled bool

	collector := f.newCollector()
	collector.OnHTML("table tbody tr", func(e *colly.HTMLElement) {
		// Проверяем контекст только один раз в начале обработки
		if contextCancelled {
			return
		}

		select {
		case <-ctx.Done():
			if !contextCancelled {
				contextCancelled = true
				f.logger.Debug("HTML processing cancelled due to context cancellation",
					zap.String("url", e.Request.URL.String()),
					zap.Error(ctx.Err()))
			}
			return
		default:
			rowCount++
			// Получаем HTML всей строки <tr>
			rowHTML, _ := e.DOM.Html()
			f.collectArtistBlock(rowHTML, artists, &artistBlocks, &mu, rowCount)
		}
	})

	collector.OnError(func(r *colly.Response, err error) {
		f.logger.Error("Failed to scrape page", zap.String("url", r.Request.URL.String()), zap.Error(err))
	})

	// Используем retry механизм для надежности
	retryConfig := RetryConfig{
		MaxRetries:        f.config.RetryConfig.MaxRetries,
		InitialDelay:      f.config.RetryConfig.InitialDelay,
		MaxDelay:          f.config.RetryConfig.MaxDelay,
		BackoffMultiplier: f.config.RetryConfig.BackoffMultiplier,
	}

	err := WithRetry(ctx, f.logger, retryConfig, func() error {
		return collector.Visit(url)
	})

	if err != nil {
		if ctx.Err() != nil {
			f.logger.Debug("ParseMonthlyPage cancelled due to context cancellation", zap.Error(ctx.Err()))
			return nil, ctx.Err()
		}
		f.logger.Error("Failed to visit page after retries", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("failed to visit page after retries: %w", err)
	}
	collector.Wait()

	if len(artistBlocks) == 0 {
		f.logger.Info("No artist blocks found for processing", zap.String("month", month), zap.String("year", year))
		return []Release{}, nil
	}

	f.logger.Info("Processing artist blocks with smart parsing", zap.Int("blocks_count", len(artistBlocks)))

	// Умный парсинг: пытаемся парсить каждый блок самостоятельно
	var smartParsedReleases []ParsedRelease
	var llmBlocks []string

	for i, block := range artistBlocks {
		// 1. Собираем RAW блоки (уже есть в block.HTML)

		// 2. Очищаем RAW блоки чистилкой
		cleanedHTML := cleanHTMLBlock(block.HTML)

		// 3. Проверяем их на "простоту" (простые комбинации)
		isSimple := IsSimpleCase(cleanedHTML, f.logger)
		if isSimple {
			// Простой случай - разбираем релиз и сохраняем
			result, err := ExtractSimpleRelease(cleanedHTML, month, year, f.logger)
			if err != nil {
				f.logger.Info("Simple extraction failed, will use LLM",
					zap.Int("block", i+1),
					zap.Error(err))
			}

			if result != nil && result.Success {
				// Умный парсинг успешен
				smartParsedReleases = append(smartParsedReleases, result.Releases...)
				f.logger.Info("Simple parsing successful",
					zap.Int("block", i+1),
					zap.Int("releases", len(result.Releases)))
				continue // Переходим к следующему блоку
			} else {
				f.logger.Info("Simple parsing failed",
					zap.Int("block", i+1),
					zap.Bool("result_nil", result == nil),
					zap.Bool("success", result != nil && result.Success),
					zap.String("error", func() string {
						if result != nil {
							return result.Error
						}
						return "result is nil"
					}()))
			}
		}

		// 4. Если блок "сложный" (простые комбинации не найдены), откладываем блок в батч
		llmBlocks = append(llmBlocks, cleanedHTML)
		f.logger.Debug("Block added to LLM queue",
			zap.Int("block", i+1),
			zap.String("reason", "complex case or extraction failed"))
	}

	// Парсим оставшиеся блоки через LLM
	var llmParsedReleases []ParsedRelease
	if len(llmBlocks) > 0 && f.llmClient != nil {
		f.logger.Info("Processing remaining blocks with LLM", zap.Int("blocks_count", len(llmBlocks)))

		llmReleases, err := f.llmParseBlocks(ctx, llmBlocks, month, year)
		if err != nil {
			f.logger.Error("Failed to parse blocks with LLM", zap.Error(err))
			return nil, fmt.Errorf("failed to parse blocks with LLM: %w", err)
		}
		llmParsedReleases = llmReleases
	} else if len(llmBlocks) > 0 {
		f.logger.Warn("Blocks need LLM processing but LLM not available", zap.Int("blocks_count", len(llmBlocks)))
		return nil, fmt.Errorf("LLM client not available for processing %d blocks", len(llmBlocks))
	}

	// Объединяем результаты
	allParsedReleases := append(smartParsedReleases, llmParsedReleases...)

	// Конвертируем в Release
	var allReleases []Release
	for _, parsedRelease := range allParsedReleases {
		// Преобразуем дату в нужный формат
		parsedDate, err := model.FormatDateWithYear(parsedRelease.Date, year, f.logger)
		if err != nil {
			f.logger.Error("Failed to parse date", zap.String("date", parsedRelease.Date), zap.Error(err))
			continue
		}

		// Проверяем, что дата соответствует месяцу
		partsDate := strings.Split(parsedDate, ".")
		if len(partsDate) != 3 || partsDate[1] != monthNum {
			f.logger.Debug("Date does not match month", zap.String("date", parsedDate), zap.String("month_num", monthNum))
			continue
		}

		// Создаем релиз
		release := Release{
			Date:       parsedDate,
			TimeMSK:    time.Now().Format("15:04"),
			Artist:     parsedRelease.Artist,
			AlbumName:  parsedRelease.Album,
			TitleTrack: parsedRelease.Track,
			MV:         parsedRelease.YouTubeURL,
		}

		allReleases = append(allReleases, release)

		f.logger.Info("Added release",
			zap.String("artist", parsedRelease.Artist),
			zap.String("date", parsedDate),
			zap.String("track", parsedRelease.Track),
			zap.String("album", parsedRelease.Album),
			zap.String("youtube", parsedRelease.YouTubeURL))
	}

	f.logger.Info("Successfully parsed releases",
		zap.String("month", month),
		zap.String("year", year),
		zap.Int("smart_parsed", len(smartParsedReleases)),
		zap.Int("llm_parsed", len(llmParsedReleases)),
		zap.Int("total_releases", len(allReleases)))

	return allReleases, nil
}
