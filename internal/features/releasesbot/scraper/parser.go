package scraper

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gemfactory/internal/features/releasesbot/release"
	"gemfactory/internal/features/releasesbot/releasefmt"
	"gemfactory/pkg/config"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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
							return
						}
					}
					artist = strings.TrimSpace(artist)
					artistKey := strings.ToLower(artist)
					if _, ok := whitelist[artistKey]; !ok {
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
							logger.Debug("No details extracted for artist", zap.String("artist", artist))
						}
						return
					}

					var events [][]string
					var eventStartIndices []int
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
							if strings.Contains(line, ":") {
								parts := strings.SplitN(line, ":", 2)
								if len(parts) == 2 {
									datePart := strings.TrimSpace(parts[0])
									parsedDate, err := releasefmt.FormatDate(datePart, logger)
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

					for idx, eventLines := range events {
						var parsedDate string
						if isDate {
							parts := strings.SplitN(eventLines[0], ":", 2)
							if len(parts) != 2 {
								continue
							}
							datePart := strings.TrimSpace(parts[0])
							var err error
							parsedDate, err = releasefmt.FormatDate(datePart, logger)
							if err != nil {
								logger.Error("Failed to parse date in event", zap.String("dateText", datePart), zap.Error(err))
								continue
							}
						} else {
							var err error
							parsedDate, err = releasefmt.FormatDate(dateText, logger)
							if err != nil {
								logger.Error("Failed to parse date in event", zap.String("dateText", dateText), zap.Error(err))
								continue
							}
						}

						partsDate := strings.Split(parsedDate, ".")
						if len(partsDate) != 3 || partsDate[1] != monthNum {
							continue
						}

						// Извлекаем альбом, трек и ссылку
						albumName := ExtractAlbumName(eventLines, 0, len(eventLines), logger)
						trackName := ExtractTrackName(eventLines, 0, len(eventLines), logger)
						startIndex := eventStartIndices[idx]
						mv := ExtractYouTubeLinkFromEvent(e, startIndex, startIndex+len(eventLines), logger)

						// Проверяем наличие события
						hasEvent := false
						for _, line := range eventLines {
							lowerLine := strings.ToLower(line)
							if strings.HasPrefix(lowerLine, "album:") ||
								strings.HasPrefix(lowerLine, "ost:") ||
								strings.HasPrefix(lowerLine, "title track:") ||
								strings.Contains(lowerLine, "pre-release") ||
								strings.Contains(lowerLine, "release") ||
								strings.Contains(lowerLine, "mv release") {
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
					mu.Unlock()
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

			collector.OnError(func(r *colly.Response, err error) {
				select {
				case <-ctx.Done():
					logger.Warn("Error processing stopped due to context cancellation", zap.String("url", r.Request.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					logger.Error("Failed to scrape page", zap.String("url", r.Request.URL.String()), zap.Error(err))
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
					logger.Debug("Completed scraping page", zap.String("url", url), zap.Int("release_count", len(artistReleases)))
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

// ExtractYouTubeLinkFromEvent extracts YouTube link from an event
func ExtractYouTubeLinkFromEvent(e *colly.HTMLElement, startIndex, endIndex int, logger *zap.Logger) string {
	var lastLink string
	var currentIndex int
	var allLinks []string

	if startIndex < 0 {
		startIndex = 0
	}

	e.ForEach("td.has-text-align-left", func(_ int, s *colly.HTMLElement) {
		s.DOM.Contents().Each(func(i int, node *goquery.Selection) {
			if node.Is("br") {
				currentIndex++
			} else if currentIndex >= startIndex && currentIndex < endIndex {
				if node.Is("a[href^='https://youtu']") {
					link := node.AttrOr("href", "")
					if !strings.HasPrefix(link, "https://www.youtube.com/@") && !strings.HasPrefix(link, "https://youtube.com/@") {
						allLinks = append(allLinks, link)
						lastLink = link
					}
				}
			}
		})
	})

	if len(allLinks) > 0 {
		return releasefmt.CleanLink(lastLink, logger)
	}
	return ""
}

// ExtractAlbumName extracts album name from lines
func ExtractAlbumName(lines []string, startIndex, endIndex int, logger *zap.Logger) string {
	for i := startIndex; i < endIndex && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(strings.ToLower(line), "album:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "album:"))
		} else if strings.HasPrefix(strings.ToLower(line), "ost:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ost:"))
		}
	}
	return "N/A"
}

// ExtractTrackName extracts track name from lines
func ExtractTrackName(lines []string, startIndex, endIndex int, logger *zap.Logger) string {
	for i := startIndex; i < endIndex && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		line = strings.ReplaceAll(line, "‘", "'")
		line = strings.ReplaceAll(line, "’", "'")
		line = strings.ReplaceAll(line, "“", "\"")
		line = strings.ReplaceAll(line, "”", "\"")

		lowerLine := strings.ToLower(line)
		var trackName string
		if strings.HasPrefix(lowerLine, "title track:") {
			trackName = strings.TrimSpace(strings.TrimPrefix(line, "title track:"))
		} else if strings.Contains(lowerLine, "release") || strings.Contains(lowerLine, "pre-release") || strings.Contains(lowerLine, "mv release") {
			trackName = line
		} else {
			continue
		}

		startDouble := strings.Index(trackName, "\"")
		endDouble := strings.LastIndex(trackName, "\"")
		startSingle := strings.Index(trackName, "'")
		endSingle := strings.LastIndex(trackName, "'")

		if startDouble != -1 && endDouble != -1 && startDouble < endDouble {
			cleaned := trackName[startDouble+1 : endDouble]
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
				return cleaned
			}
		} else if startSingle != -1 && endSingle != -1 && startSingle < endSingle {
			cleaned := trackName[startSingle+1 : endSingle]
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
				return cleaned
			}
		}
	}
	return "N/A"
}
