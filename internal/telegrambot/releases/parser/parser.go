package parser

import (
	"context"
	"fmt"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/releasefmt"
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
	"go.uber.org/zap/zapcore"
)

// NewCollector creates a new Colly collector with configured settings
func NewCollector(config *config.Config, logger *zap.Logger) *colly.Collector {
	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Устанавливаем HTTP-клиент с таймаутом
	collector.WithTransport(&http.Transport{
		ResponseHeaderTimeout: 60 * time.Second,
		DisableKeepAlives:     true,
	})

	if err := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       config.RequestDelay * 2, // Задержка 6s при REQUEST_DELAY=3s
		RandomDelay: config.RequestDelay,
	}); err != nil {
		logger.Error("Failed to set collector limit", zap.Error(err))
	}

	// Реализуем повторы вручную через OnError
	collector.OnError(func(r *colly.Response, err error) {
		maxRetries := config.MaxRetries
		retries := r.Request.Ctx.GetAny("retries")
		retryCount, ok := retries.(int)
		if !ok {
			retryCount = 0
		}

		if retryCount < maxRetries {
			retryCount++
			r.Request.Ctx.Put("retries", retryCount)
			logger.Warn("Retrying request", zap.String("url", r.Request.URL.String()), zap.Int("retry", retryCount), zap.Error(err))
			if err := r.Request.Retry(); err != nil {
				logger.Error("Failed to retry request", zap.String("url", r.Request.URL.String()), zap.Int("retry", retryCount), zap.Error(err))
			}
			return
		}

		logger.Error("Request failed after max retries", zap.String("url", r.Request.URL.String()), zap.Int("retries", retryCount), zap.Error(err))
	})

	// Добавляем случайный User-Agent
	extensions.RandomUserAgent(collector)

	return collector
}

// ParseMonthlyPageWithContext parses a monthly schedule page with context
func ParseMonthlyPageWithContext(ctx context.Context, url string, whitelist map[string]struct{}, targetMonth string, config *config.Config, logger *zap.Logger) ([]release.Release, error) {
	if logger.Core().Enabled(zapcore.DebugLevel) {
		logger.Debug("Starting ParseMonthlyPageWithContext", zap.String("url", url), zap.String("targetMonth", targetMonth))
	}
	monthNum, ok := release.MonthToNumber[strings.ToLower(targetMonth)]
	if !ok {
		logger.Error("Unknown month", zap.String("month", targetMonth))
		return nil, fmt.Errorf("unknown month: %s", targetMonth)
	}

	var allReleases []release.Release
	var err error

	// Пробуем до трёх раз при таймаутах
	for attempt := 0; attempt < 3; attempt++ {
		select {
		case <-ctx.Done():
			logger.Warn("ParseMonthlyPage stopped due to context cancellation", zap.Error(ctx.Err()))
			return nil, ctx.Err()
		default:
			if logger.Core().Enabled(zapcore.DebugLevel) {
				logger.Debug("ParseMonthlyPage attempt", zap.Int("attempt", attempt+1), zap.String("url", url))
			}
			artistReleases := make(map[string][]release.Release)
			collector := NewCollector(config, logger)
			var rowCount int
			var mu sync.Mutex

			collector.OnHTML("tr", func(e *colly.HTMLElement) {
				select {
				case <-ctx.Done():
					logger.Warn("OnHTML processing stopped due to context cancellation", zap.String("url", e.Request.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					mu.Lock()
					rowCount++
					mu.Unlock()

					dateText := e.ChildText("td.has-text-align-right mark")
					if dateText == "" {
						if logger.Core().Enabled(zapcore.DebugLevel) {
							logger.Debug("No date found in row", zap.Int("row", rowCount))
						}
						return
					}

					timeText := e.ChildText("td.has-text-align-right")
					timeKST := ""
					if strings.Contains(timeText, "at") {
						var err error
						timeKST, err = releasefmt.FormatTimeKST(timeText, logger)
						if err != nil {
							timeKST = ""
						}
					}
					timeMSK, err := releasefmt.ConvertKSTtoMSK(timeKST, logger)
					if err != nil {
						timeMSK = ""
					}

					artist := e.ChildText("td.has-text-align-left strong mark")
					if artist == "" {
						artist = e.ChildText("td.has-text-align-left strong")
						if artist == "" {
							if logger.Core().Enabled(zapcore.DebugLevel) {
								logger.Debug("No artist found in row", zap.Int("row", rowCount))
							}
							return
						}
					}
					artist = strings.TrimSpace(artist)
					// Проверяем наличие слова POSTPONED
					detailsText := e.ChildText("td.has-text-align-left")
					if strings.Contains(strings.ToLower(detailsText), "postponed") {
						if logger.Core().Enabled(zapcore.DebugLevel) {
							logger.Debug("Event postponed, skipping", zap.String("artist", artist), zap.Int("row", rowCount))
						}
						return
					}

					artistKey := strings.ToLower(artist)
					if _, ok := whitelist[artistKey]; !ok {
						if logger.Core().Enabled(zapcore.DebugLevel) {
							logger.Debug("Artist not in whitelist", zap.String("artist", artist), zap.Int("row", rowCount))
						}
						return
					}

					var detailsLines []string
					var currentLine strings.Builder
					e.ForEach("td.has-text-align-left", func(_ int, s *colly.HTMLElement) {
						s.DOM.Contents().Each(func(_ int, node *goquery.Selection) {
							if node.Is("br") {
								if currentLine.Len() > 0 {
									detailsLines = append(detailsLines, strings.TrimSpace(currentLine.String()))
									currentLine.Reset()
								}
							} else if text := strings.TrimSpace(node.Text()); text != "" {
								if currentLine.Len() > 0 {
									currentLine.WriteString(" ")
								}
								currentLine.WriteString(text)
							}
						})
						if currentLine.Len() > 0 {
							detailsLines = append(detailsLines, strings.TrimSpace(currentLine.String()))
							currentLine.Reset()
						}
					})

					if len(detailsLines) < 1 {
						if logger.Core().Enabled(zapcore.DebugLevel) {
							logger.Debug("No details extracted for artist", zap.String("artist", artist), zap.Int("row", rowCount))
						}
						return
					}

					var events [][]string
					var eventStartIndices []int
					if len(detailsLines) > 1 {
						firstLineAfterArtist := detailsLines[1]
						isDate := false
						for _, month := range release.Months {
							if strings.HasPrefix(strings.ToLower(firstLineAfterArtist), month) {
								isDate = true
								break
							}
						}

						if isDate {
							currentEvent := []string{}
							currentIndex := 1
							for i := 1; i < len(detailsLines); i++ {
								line := detailsLines[i]
								// Проверяем, является ли строка началом нового события (валидная дата)
								if strings.Contains(line, ":") {
									parts := strings.SplitN(line, ":", 2)
									if len(parts) == 2 {
										datePart := strings.TrimSpace(parts[0])
										// Проверяем, начинается ли строка с месяца и является ли она валидной датой
										for _, month := range release.Months {
											if strings.HasPrefix(strings.ToLower(datePart), month) {
												if _, err := releasefmt.FormatDate(datePart, logger); err == nil {
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
													break
												}
											}
										}
										if i < len(detailsLines)-1 && len(currentEvent) == 0 {
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
							// Если первая строка не дата, считаем все строки одним событием
							eventLines := detailsLines[1:]
							events = append(events, eventLines)
							eventStartIndices = append(eventStartIndices, 1)
						}
					} else {
						events = append(events, detailsLines[1:])
						eventStartIndices = append(eventStartIndices, 1)
					}

					for idx, eventLines := range events {
						var parsedDate string
						if len(eventLines) > 0 && strings.Contains(eventLines[0], ":") {
							parts := strings.SplitN(eventLines[0], ":", 2)
							if len(parts) == 2 {
								datePart := strings.TrimSpace(parts[0])
								// Проверяем, является ли строка валидной датой
								for _, month := range release.Months {
									if strings.HasPrefix(strings.ToLower(datePart), month) {
										var err error
										parsedDate, err = releasefmt.FormatDate(datePart, logger)
										if err != nil {
											logger.Error("Failed to parse date in event", zap.String("dateText", datePart), zap.Error(err))
											continue
										}
										break
									}
								}
							}
						}
						// Если дата не найдена в строке события, используем dateText из столбца
						if parsedDate == "" {
							var err error
							parsedDate, err = releasefmt.FormatDate(dateText, logger)
							if err != nil {
								logger.Error("Failed to parse date in event", zap.String("dateText", dateText), zap.Error(err))
								continue
							}
						}

						partsDate := strings.Split(parsedDate, ".")
						if len(partsDate) != 3 || partsDate[1] != monthNum {
							if logger.Core().Enabled(zapcore.DebugLevel) {
								logger.Debug("Date does not match target month", zap.String("parsedDate", parsedDate), zap.String("monthNum", monthNum))
							}
							continue
						}

						// Проверяем наличие события
						hasEvent := false
						for _, line := range eventLines {
							lowerLine := strings.ToLower(line)
							if strings.Contains(lowerLine, "teaser") || strings.Contains(lowerLine, "poster") {
								continue
							}
							if strings.Contains(lowerLine, "album") ||
								strings.Contains(lowerLine, "ost") ||
								strings.Contains(lowerLine, "title track") ||
								strings.Contains(lowerLine, "pre-release") ||
								(strings.Contains(lowerLine, "release") && strings.Contains(lowerLine, "mv")) ||
								strings.Contains(lowerLine, "mini album") ||
								strings.Contains(lowerLine, "special mini album") {
								hasEvent = true
								break
							}
						}

						if !hasEvent {
							if logger.Core().Enabled(zapcore.DebugLevel) {
								logger.Debug("No event found for release", zap.String("artist", artist), zap.String("date", parsedDate), zap.Strings("eventLines", eventLines))
							}
							continue
						}

						// Извлекаем альбом, трек и ссылку
						albumName := ExtractAlbumName(eventLines, 0, len(eventLines), logger)
						trackName := ExtractTrackName(eventLines, 0, len(eventLines), logger)
						startIndex := eventStartIndices[idx]
						mv := ExtractYouTubeLinkFromEvent(e, startIndex, startIndex+len(eventLines), logger)

						// Создаём релиз
						release := release.Release{
							Date:       parsedDate,
							TimeMSK:    timeMSK,
							Artist:     artist,
							AlbumName:  albumName,
							TitleTrack: trackName,
							MV:         mv,
						}
						key := fmt.Sprintf("%s-%s", strings.ToLower(artist), parsedDate)
						mu.Lock()
						artistReleases[key] = append(artistReleases[key], release)
						mu.Unlock()
					}

					mu.Lock()
					totalReleases := 0
					for _, releases := range artistReleases {
						totalReleases += len(releases)
					}
					if logger.Core().Enabled(zapcore.DebugLevel) {
						logger.Debug("Processed row", zap.Int("row", rowCount), zap.String("artist", artist), zap.Int("releases", totalReleases))
					}
					mu.Unlock()
				}
			})

			collector.OnError(func(r *colly.Response, err error) {
				select {
				case <-ctx.Done():
					logger.Warn("Error processing stopped due to context cancellation", zap.String("url", r.Request.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					logger.Error("Failed to scrape page", zap.String("url", r.Request.URL.String()), zap.Error(err))
				}
			})

			collector.OnRequest(func(r *colly.Request) {
				select {
				case <-ctx.Done():
					logger.Warn("OnRequest stopped due to context cancellation", zap.String("url", r.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					if logger.Core().Enabled(zapcore.DebugLevel) {
						logger.Debug("Visiting URL", zap.String("url", r.URL.String()))
					}
					r.Ctx.Put("start_time", time.Now())
				}
			})

			collector.OnResponse(func(r *colly.Response) {
				select {
				case <-ctx.Done():
					logger.Warn("OnResponse stopped due to context cancellation", zap.String("url", r.Request.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					startTime, _ := r.Ctx.GetAny("start_time").(time.Time)
					if logger.Core().Enabled(zapcore.DebugLevel) {
						logger.Debug("Received response", zap.String("url", r.Request.URL.String()), zap.Duration("duration", time.Since(startTime)))
					}
				}
			})

			requestCtx, requestCancel := context.WithTimeout(ctx, 90*time.Second)
			defer requestCancel()

			err = collector.Visit(url)
			collector.Wait()

			select {
			case <-requestCtx.Done():
				logger.Warn("Scraping page timed out", zap.String("url", url), zap.Error(requestCtx.Err()))
				if attempt < 2 {
					logger.Info("Retrying ParseMonthlyPage due to timeout", zap.Int("attempt", attempt+1))
					artistReleases = make(map[string][]release.Release)
					rowCount = 0
					continue
				}
				return nil, requestCtx.Err()
			default:
				if err != nil {
					logger.Error("Failed to visit page", zap.String("url", url), zap.Error(err))
					if attempt < 2 {
						logger.Info("Retrying ParseMonthlyPage due to error", zap.Int("attempt", attempt+1), zap.Error(err))
						artistReleases = make(map[string][]release.Release)
						rowCount = 0
						continue
					}
					return nil, fmt.Errorf("failed to visit page: %v", err)
				}
				if logger.Core().Enabled(zapcore.DebugLevel) {
					logger.Debug("Completed scraping page", zap.String("url", url), zap.Int("release_count", len(artistReleases)), zap.Int("total_rows", rowCount))
				}
			}

			for _, releases := range artistReleases {
				sort.Slice(releases, func(i, j int) bool {
					dateI, _ := time.Parse(release.DateFormat, releases[i].Date)
					dateJ, _ := time.Parse(release.DateFormat, releases[j].Date)
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
					if rel.TitleTrack != "N/A" && rel.MV != "" {
						bestRelease = rel
						break
					} else if rel.TitleTrack != "N/A" && bestRelease.TitleTrack == "N/A" {
						bestRelease = rel
					} else if rel.MV != "" && bestRelease.MV == "" {
						bestRelease = rel
					}
				}

				allReleases = append(allReleases, bestRelease)
			}

			sort.Slice(allReleases, func(i, j int) bool {
				dateI, _ := time.Parse(release.DateFormat, allReleases[i].Date)
				dateJ, _ := time.Parse(release.DateFormat, allReleases[j].Date)
				return dateI.Before(dateJ)
			})

			return allReleases, nil
		}
	}

	return allReleases, nil
}
