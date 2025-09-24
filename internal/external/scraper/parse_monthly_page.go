package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// cleanHTMLBlock очищает HTML блок по описанному паттерну
func cleanHTMLBlock(html string) string {
	// 1. Удаляем все атрибуты из тегов
	html = regexp.MustCompile(`\s+(class|style|data-[^=]+)="[^"]*"`).ReplaceAllString(html, "")

	// 2. Удаляем теги форматирования, оставляя содержимое
	html = regexp.MustCompile(`</?(mark|strong|span)[^>]*>`).ReplaceAllString(html, "")

	// 3. Удаляем ненужные ссылки (все кроме YouTube)
	allLinksRegex := regexp.MustCompile(`<a[^>]*href="[^"]*"[^>]*>.*?</a>`)
	html = allLinksRegex.ReplaceAllStringFunc(html, func(match string) string {
		// Проверяем, является ли это YouTube ссылкой
		if strings.Contains(match, "youtube.com") || strings.Contains(match, "youtu.be") {
			// Извлекаем URL из YouTube ссылки
			urlRegex := regexp.MustCompile(`href="(https?://[^"]*)"`)
			urlMatch := urlRegex.FindStringSubmatch(match)
			if len(urlMatch) > 1 {
				return fmt.Sprintf(`<a href="%s"></a>`, urlMatch[1])
			}
			return match
		}
		return "" // Удаляем все остальные ссылки
	})

	// 4. Удаляем текст после артиста (в скобках)
	html = regexp.MustCompile(`\s*\([^)]*\)`).ReplaceAllString(html, "")

	// 5. Заменяем HTML entities
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")

	// 6. Заменяем <br> на переносы строк
	html = regexp.MustCompile(`<br\s*/?>`).ReplaceAllString(html, "\n")

	// 7. Удаляем множественные пробелы и переносы
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")
	html = regexp.MustCompile(`\n\s*`).ReplaceAllString(html, "\n")

	return strings.TrimSpace(html)
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
			// Получаем HTML блока <td>
			blockHTML, _ := e.DOM.Html()
			f.collectArtistBlock(blockHTML, monthNum, year, artists, &artistBlocks, &mu, rowCount)
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

	// Обрабатываем все собранные блоки через LLM
	if len(artistBlocks) > 0 && f.llmClient != nil {
		f.logger.Info("Processing artist blocks with LLM", zap.Int("blocks_count", len(artistBlocks)))

		// Собираем очищенные HTML блоки, разделяя их точкой с запятой
		var htmlBlocks []string
		for i, block := range artistBlocks {
			cleanedHTML := cleanHTMLBlock(block.HTML)
			htmlBlocks = append(htmlBlocks, fmt.Sprintf("BLOCK %d: %s", i+1, cleanedHTML))

			// Показываем пример первого блока
			if i == 0 {
				f.logger.Info("EXAMPLE ORIGINAL HTML BLOCK",
					zap.String("original_html", block.HTML))
				f.logger.Info("EXAMPLE CLEANED HTML BLOCK",
					zap.String("cleaned_html", cleanedHTML))
			}
		}

		// Объединяем все блоки через точку с запятой
		allBlocksText := strings.Join(htmlBlocks, "; ")

		// Отправляем батч в LLM
		llmResponse, err := f.llmClient.ParseMultiRelease(ctx, allBlocksText)
		if err != nil {
			f.logger.Error("Failed to parse artist blocks with LLM", zap.Error(err))
			return nil, fmt.Errorf("failed to parse artist blocks with LLM: %w", err)
		}

		// Обрабатываем ответ LLM и создаем релизы
		var allReleases []Release
		for _, releaseData := range llmResponse.Releases {
			// Преобразуем дату в нужный формат
			parsedDate, err := FormatDateWithYear(releaseData.Date, year, f.logger)
			if err != nil {
				f.logger.Error("Failed to parse date from LLM", zap.String("date", releaseData.Date), zap.Error(err))
				continue
			}

			// Проверяем, что дата соответствует месяцу
			partsDate := strings.Split(parsedDate, ".")
			if len(partsDate) != 3 || partsDate[1] != monthNum {
				f.logger.Debug("Date from LLM does not match month", zap.String("date", parsedDate), zap.String("month_num", monthNum))
				continue
			}

			// Создаем релиз
			release := Release{
				Date:       parsedDate,
				TimeMSK:    time.Now().Format("15:04"),
				Artist:     releaseData.Artist,
				AlbumName:  releaseData.Album,
				TitleTrack: releaseData.Track,
				MV:         releaseData.YouTubeURL,
			}

			allReleases = append(allReleases, release)

			f.logger.Info("Added release from LLM",
				zap.String("artist", releaseData.Artist),
				zap.String("date", parsedDate),
				zap.String("track", releaseData.Track),
				zap.String("album", releaseData.Album),
				zap.String("youtube", releaseData.YouTubeURL))
		}

		f.logger.Info("Successfully parsed releases with LLM",
			zap.String("month", month),
			zap.String("year", year),
			zap.Int("total_releases", len(allReleases)))

		return allReleases, nil
	} else if len(artistBlocks) > 0 {
		f.logger.Warn("Artist blocks found but LLM not available", zap.Int("blocks_count", len(artistBlocks)))
		return nil, fmt.Errorf("LLM client not available for processing %d artist blocks", len(artistBlocks))
	}

	f.logger.Info("No artist blocks found for processing", zap.String("month", month), zap.String("year", year))
	return []Release{}, nil
}
