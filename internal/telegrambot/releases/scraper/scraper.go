package scraper

import (
	"context"
	"fmt"
	"gemfactory/internal/telegrambot/releases/middleware"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/service"
	"gemfactory/pkg/config"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"go.uber.org/zap"
)

// fetcherImpl implements the Fetcher interface
type fetcherImpl struct {
	config     *config.Config
	logger     *zap.Logger
	httpClient *http.Client
}

// NewFetcher creates a new Fetcher instance
func NewFetcher(config *config.Config, logger *zap.Logger) Fetcher {
	httpConfig := HTTPClientConfig{
		MaxIdleConns:          config.HTTPClientConfig.MaxIdleConns,
		MaxIdleConnsPerHost:   config.HTTPClientConfig.MaxIdleConnsPerHost,
		IdleConnTimeout:       config.HTTPClientConfig.IdleConnTimeout,
		TLSHandshakeTimeout:   config.HTTPClientConfig.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.HTTPClientConfig.ResponseHeaderTimeout,
		DisableKeepAlives:     config.HTTPClientConfig.DisableKeepAlives,
	}

	httpClient := NewHTTPClient(httpConfig, logger)

	return &fetcherImpl{
		config:     config,
		logger:     logger,
		httpClient: httpClient,
	}
}

// FetchMonthlyLinks retrieves links to monthly schedules
func (f *fetcherImpl) FetchMonthlyLinks(ctx context.Context, months []string) ([]string, error) {
	var links []string
	uniqueLinks := make(map[string]struct{})
	var mu sync.Mutex

	collector := f.newCollector()
	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		select {
		case <-ctx.Done():
			f.logger.Warn("OnHTML processing stopped", zap.String("url", e.Request.URL.String()), zap.Error(ctx.Err()))
			return
		default:
			link := e.Attr("href")
			if strings.Contains(link, "kpop-comeback-schedule-") &&
				strings.Contains(link, "https://kpopofficial.com/") &&
				strings.Contains(link, release.CurrentYear()) {
				for _, month := range months {
					if strings.Contains(strings.ToLower(link), strings.ToLower(month)) {
						mu.Lock()
						if _, exists := uniqueLinks[link]; !exists {
							uniqueLinks[link] = struct{}{}
							links = append(links, link)
						}
						mu.Unlock()
					}
				}
			}
		}
	})

	collector.OnError(func(r *colly.Response, err error) {
		f.logger.Error("Failed to scrape links",
			zap.String("url", r.Request.URL.String()),
			zap.Error(err),
			zap.Int("status_code", r.StatusCode))
	})

	url := "https://kpopofficial.com/category/kpop-comeback-schedule/"

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
		f.logger.Error("Failed to visit main page after retries", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("failed to visit main page after retries: %w", err)
	}
	collector.Wait()

	f.logger.Info("Fetched links", zap.Int("link_count", len(links)))
	return links, nil
}

// ParseMonthlyPage parses a monthly schedule page
func (f *fetcherImpl) ParseMonthlyPage(ctx context.Context, url, month string, whitelist map[string]struct{}) ([]release.Release, error) {
	cfg := release.NewConfig()
	monthNum, ok := cfg.MonthToNumber(strings.ToLower(month))
	if !ok {
		f.logger.Error("Unknown month", zap.String("month", month))
		return nil, fmt.Errorf("unknown month: %s", month)
	}

	artistReleases := make(map[string][]release.Release)
	allReleases := make([]release.Release, 0, len(artistReleases))
	var mu sync.Mutex
	rowCount := 0

	collector := f.newCollector()
	collector.OnHTML("tr", func(e *colly.HTMLElement) {
		select {
		case <-ctx.Done():
			f.logger.Warn("OnHTML processing stopped", zap.String("url", e.Request.URL.String()), zap.Error(ctx.Err()))
			return
		default:
			rowCount++
			f.extractRow(ctx, e, monthNum, whitelist, artistReleases, &mu, rowCount)
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
		f.logger.Error("Failed to visit page after retries", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("failed to visit page after retries: %w", err)
	}
	collector.Wait()

	for _, releases := range artistReleases {
		if len(releases) == 0 {
			continue
		}
		sort.Slice(releases, func(i, j int) bool {
			dateI, _ := time.Parse(cfg.DateFormat(), releases[i].Date)
			dateJ, _ := time.Parse(cfg.DateFormat(), releases[j].Date)
			return dateI.Before(dateJ)
		})

		var bestRelease release.Release
		found := false
		for _, rel := range releases {
			if !found {
				bestRelease = rel
				found = true
				continue
			}

			switch {
			case rel.TitleTrack != "N/A" && rel.MV != "":
				bestRelease = rel
				goto foundBest
			case rel.TitleTrack != "N/A" && bestRelease.TitleTrack == "N/A":
				bestRelease = rel
			case rel.MV != "" && bestRelease.MV == "":
				bestRelease = rel
			}
		}
	foundBest:
		allReleases = append(allReleases, bestRelease)
	}

	sort.Slice(allReleases, func(i, j int) bool {
		dateI, _ := time.Parse(cfg.DateFormat(), allReleases[i].Date)
		dateJ, _ := time.Parse(cfg.DateFormat(), allReleases[j].Date)
		return dateI.Before(dateJ)
	})

	f.logger.Debug("Parsed releases", zap.String("month", month), zap.Int("release_count", len(allReleases)))
	return allReleases, nil
}

// newCollector creates a new Colly collector with configured middleware
func (f *fetcherImpl) newCollector() *colly.Collector {
	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Используем оптимизированный HTTP клиент
	if f.httpClient != nil {
		collector.WithTransport(f.httpClient.Transport)
	} else {
		// Fallback к стандартным настройкам
		collector.WithTransport(&http.Transport{
			ResponseHeaderTimeout: 180 * time.Second,
			DisableKeepAlives:     true,
		})
	}

	if err := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       f.config.RequestDelay * 2,
		RandomDelay: f.config.RequestDelay,
	}); err != nil {
		f.logger.Error("Failed to set collector limit", zap.Error(err))
	}

	collector.OnRequest(func(r *colly.Request) {
		r.Ctx.Put("start_time", time.Now())
		if err := middleware.WithLogging(f.logger)(r, f.logger); err != nil {
			f.logger.Error("Logging middleware failed", zap.Error(err))
		}
	})

	collector.OnResponse(func(r *colly.Response) {
		startTime, _ := r.Ctx.GetAny("start_time").(time.Time)
		f.logger.Debug("Scraped page",
			zap.String("url", r.Request.URL.String()),
			zap.Int("status_code", r.StatusCode),
			zap.Duration("duration", time.Since(startTime)))
	})

	extensions.RandomUserAgent(collector)
	return collector
}

// extractRow extracts release data from a table row
func (f *fetcherImpl) extractRow(_ context.Context, e *colly.HTMLElement, monthNum string, whitelist map[string]struct{}, artistReleases map[string][]release.Release, mu *sync.Mutex, rowCount int) {
	cfg := release.NewConfig()
	dateText := e.ChildText("td.has-text-align-right mark")
	if dateText == "" {
		f.logger.Debug("No date found for row", zap.Int("row", rowCount))
		return
	}

	timeText := e.ChildText("td.has-text-align-right")
	timeKST := ""
	if strings.Contains(timeText, "at") {
		var err error
		timeKST, err = service.FormatTimeKST(timeText, f.logger)
		if err != nil {
			f.logger.Debug("Failed to format KST time", zap.String("time", timeText), zap.Error(err))
			timeKST = ""
		}
	}
	timeMSK, err := service.ConvertKSTtoMSK(timeKST, f.logger)
	if err != nil {
		f.logger.Debug("Failed to convert KST to MSK", zap.String("time", timeKST), zap.Error(err))
		timeMSK = ""
	}

	artist := e.ChildText("td.has-text-align-left strong mark")
	if artist == "" {
		artist = e.ChildText("td.has-text-align-left strong")
	}
	artist = strings.TrimSpace(artist)
	artistKey := strings.ToLower(artist)
	f.logger.Debug("Found artist", zap.String("artist", artist), zap.String("artist_key", artistKey), zap.Int("row", rowCount))
	if _, ok := whitelist[artistKey]; !ok {
		f.logger.Debug("Artist not in whitelist", zap.String("artist_key", artistKey), zap.Int("row", rowCount))
		return
	}

	var detailsLines []string
	e.ForEach("td.has-text-align-left", func(_ int, s *colly.HTMLElement) {
		var currentLine []string
		s.DOM.Contents().Each(func(_ int, node *goquery.Selection) {
			if node.Is("br") {
				if len(currentLine) > 0 {
					detailsLines = append(detailsLines, strings.Join(currentLine, " "))
					currentLine = nil
				}
			} else if text := strings.TrimSpace(node.Text()); text != "" {
				currentLine = append(currentLine, text)
			}
		})
		if len(currentLine) > 0 {
			detailsLines = append(detailsLines, strings.Join(currentLine, " "))
		}
	})
	f.logger.Debug("Extracted details lines", zap.Strings("details", detailsLines), zap.Int("row", rowCount))

	if len(detailsLines) < 1 {
		f.logger.Debug("No details lines found", zap.Int("row", rowCount))
		return
	}

	var events [][]string
	var eventStartIndices []int
	firstLineAfterArtist := detailsLines[1]
	isDate := false
	for _, m := range cfg.Months() {
		if strings.HasPrefix(strings.ToLower(firstLineAfterArtist), m) {
			isDate = true
			break
		}
	}
	f.logger.Debug("Checked date presence", zap.Bool("is_date", isDate), zap.String("first_line", firstLineAfterArtist), zap.Int("row", rowCount))

	if isDate {
		currentEvent := []string{}
		currentIndex := 1
		for i := 1; i < len(detailsLines); i++ {
			line := detailsLines[i]
			if strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					datePart := strings.TrimSpace(parts[0])
					parsedDate, err := service.FormatDate(datePart, f.logger)
					if err == nil && parsedDate != "" {
						if len(currentEvent) > 0 {
							events = append(events, currentEvent)
							startIndex := currentIndex
							if startIndex < 1 {
								startIndex = 1
							}
							eventStartIndices = append(eventStartIndices, startIndex)
						}
						currentEvent = []string{line}
						currentIndex = i
						f.logger.Debug("New event started", zap.String("date", parsedDate), zap.Int("index", i), zap.Int("row", rowCount))
						continue
					}
				}
			}
			currentEvent = append(currentEvent, line)
		}
		if len(currentEvent) > 0 {
			events = append(events, currentEvent)
			startIndex := currentIndex
			if startIndex < 1 {
				startIndex = 1
			}
			eventStartIndices = append(eventStartIndices, startIndex)
		}
	} else {
		eventLines := detailsLines[1:]
		events = append(events, eventLines)
		eventStartIndices = append(eventStartIndices, 1)
	}
	f.logger.Debug("Extracted events", zap.Int("event_count", len(events)), zap.Int("row", rowCount))

	for idx, eventLines := range events {
		var parsedDate string
		if isDate {
			parts := strings.SplitN(eventLines[0], ":", 2)
			if len(parts) != 2 {
				f.logger.Debug("Invalid event format", zap.Int("event_index", idx), zap.Int("row", rowCount))
				continue
			}
			datePart := strings.TrimSpace(parts[0])
			var err error
			parsedDate, err = service.FormatDate(datePart, f.logger)
			if err != nil {
				f.logger.Error("Failed to parse date in event", zap.String("dateText", datePart), zap.Error(err))
				continue
			}
			f.logger.Debug("Parsed event date", zap.String("date", parsedDate), zap.Int("event_index", idx), zap.Int("row", rowCount))
		} else {
			var err error
			parsedDate, err = service.FormatDate(dateText, f.logger)
			if err != nil {
				f.logger.Error("Failed to parse date in event", zap.String("dateText", dateText), zap.Error(err))
				continue
			}
			f.logger.Debug("Parsed row date", zap.String("date", parsedDate), zap.Int("event_index", idx), zap.Int("row", rowCount))
		}

		partsDate := strings.Split(parsedDate, ".")
		if len(partsDate) != 3 || partsDate[1] != monthNum {
			f.logger.Debug("Date does not match month", zap.String("date", parsedDate), zap.String("month_num", monthNum), zap.Int("event_index", idx), zap.Int("row", rowCount))
			continue
		}

		albumName := ExtractAlbumName(eventLines, 0, len(eventLines), f.logger)
		f.logger.Debug("Extracted album", zap.String("album", albumName), zap.Int("event_index", idx), zap.Int("row", rowCount))
		trackName := ExtractTrackName(eventLines, 0, len(eventLines), f.logger)
		f.logger.Debug("Extracted track", zap.String("track", trackName), zap.Int("event_index", idx), zap.Int("row", rowCount))
		startIndex := eventStartIndices[idx]
		mv := ExtractYouTubeLinkFromEvent(e, startIndex, startIndex+len(eventLines), f.logger)
		f.logger.Debug("Extracted MV", zap.String("mv", mv), zap.Int("event_index", idx), zap.Int("row", rowCount))

		hasEvent := false
		for _, line := range eventLines {
			lowerLine := strings.ToLower(line)
			if strings.HasPrefix(lowerLine, "album:") ||
				strings.HasPrefix(lowerLine, "ost:") ||
				strings.HasPrefix(lowerLine, "title track:") ||
				strings.Contains(lowerLine, "pre-release") ||
				strings.Contains(lowerLine, "release") ||
				strings.Contains(lowerLine, "mv release") ||
				strings.Contains(lowerLine, "album") {
				hasEvent = true
				break
			}
		}
		f.logger.Debug("Checked event presence", zap.Bool("has_event", hasEvent), zap.Int("event_index", idx), zap.Int("row", rowCount))

		if !hasEvent {
			continue
		}

		release := release.Release{
			Date:       parsedDate,
			TimeMSK:    timeMSK,
			Artist:     artist,
			AlbumName:  albumName,
			TitleTrack: trackName,
			MV:         mv,
		}
		f.logger.Debug("Created release",
			zap.String("artist", release.Artist),
			zap.String("date", release.Date),
			zap.String("album", release.AlbumName),
			zap.String("track", release.TitleTrack),
			zap.String("mv", release.MV),
			zap.Int("event_index", idx),
			zap.Int("row", rowCount))
		key := fmt.Sprintf("%s-%s", strings.ToLower(artist), parsedDate)
		mu.Lock()
		artistReleases[key] = append(artistReleases[key], release)
		mu.Unlock()
	}
}
