package parser

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gemfactory/formatter"
	"gemfactory/models"
	"gemfactory/utils"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"go.uber.org/zap"
)

// GetMonthlyLinks retrieves links to monthly schedules
func GetMonthlyLinks(months []string, logger *zap.Logger) ([]string, error) {
	maxRetries, delay := utils.GetCollectorConfig()
	var monthlyLinks []string
	uniqueLinks := make(map[string]struct{})
	currentYear := models.CurrentYear()

	collector := NewCollector(maxRetries, delay, logger)
	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if strings.Contains(link, "kpop-comeback-schedule-") &&
			strings.Contains(link, "https://kpopofficial.com/") &&
			strings.Contains(link, currentYear) {
			if len(months) > 0 {
				for _, month := range months {
					if strings.Contains(strings.ToLower(link), strings.ToLower(month)) {
						if _, exists := uniqueLinks[link]; !exists {
							uniqueLinks[link] = struct{}{}
							monthlyLinks = append(monthlyLinks, link)
						}
					}
				}
			} else {
				if _, exists := uniqueLinks[link]; !exists {
					uniqueLinks[link] = struct{}{}
					monthlyLinks = append(monthlyLinks, link)
				}
			}
		}
	})

	err := collector.Visit("https://kpopofficial.com/category/kpop-comeback-schedule/")
	if err != nil {
		logger.Error("Failed to visit main page", zap.Error(err))
		return nil, fmt.Errorf("failed to visit main page: %v", err)
	}

	collector.Wait()
	return monthlyLinks, nil
}

// ParseMonthlyPage parses a monthly schedule page
func ParseMonthlyPage(url string, whitelist map[string]struct{}, targetMonth string, logger *zap.Logger) ([]models.Release, error) {
	maxRetries, delay := utils.GetCollectorConfig()

	monthNum, ok := models.MonthToNumber[strings.ToLower(targetMonth)]
	if !ok {
		logger.Error("Unknown month", zap.String("month", targetMonth))
		return nil, fmt.Errorf("unknown month: %s", targetMonth)
	}

	var allReleases []models.Release
	artistReleases := make(map[string][]models.Release)
	collector := NewCollector(maxRetries, delay, logger)
	var rowCount int

	collector.OnHTML("tr", func(e *colly.HTMLElement) {
		rowCount++
		dateText := e.ChildText("td.has-text-align-right mark")
		if dateText == "" {
			return
		}

		timeText := e.ChildText("td.has-text-align-right")
		timeKST := ""
		if strings.Contains(timeText, "at") {
			var err error
			timeKST, err = formatter.FormatTimeKST(timeText, logger)
			if err != nil {
				timeKST = ""
			}
		}
		timeMSK, err := formatter.ConvertKSTtoMSK(timeKST, logger)
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
			logger.Debug("No details extracted for artist", zap.String("artist", artist)) // Добавляем лог
			return
		}

		var events [][]string
		var eventStartIndices []int
		firstLineAfterArtist := detailsLines[1]
		isDate := false
		for _, month := range models.Months {
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
						parsedDate, err := formatter.FormatDate(datePart, logger)
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
				parsedDate, err = formatter.FormatDate(datePart, logger)
				if err != nil {
					logger.Error("Failed to parse date in event", zap.String("dateText", datePart), zap.Error(err))
					continue
				}
			} else {
				var err error
				parsedDate, err = formatter.FormatDate(dateText, logger)
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
				logger.Debug("No event found for release", zap.String("artist", artist), zap.String("date", parsedDate), zap.Strings("eventLines", eventLines)) // Обновляем лог с eventLines
				continue
			}

			// Создаём релиз
			release := models.Release{
				Date:       parsedDate,
				TimeMSK:    timeMSK,
				Artist:     artist,
				AlbumName:  albumName,
				TitleTrack: trackName,
				MV:         mv,
			}
			key := fmt.Sprintf("%s-%s", strings.ToLower(artist), parsedDate)
			artistReleases[key] = append(artistReleases[key], release)
		}

		// Убираем промежуточное логирование
		totalReleases := 0
		for _, releases := range artistReleases {
			totalReleases += len(releases)
		}
		// Убираем logger.Info("Completed parsing page", ...)
	})

	if err := collector.Visit(url); err != nil {
		logger.Error("Failed to visit page", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("failed to visit page: %v", err)
	}

	collector.Wait()
	//logger.Info("Processed rows", zap.String("url", url), zap.Int("row_count", rowCount))

	for _, releases := range artistReleases {
		sort.Slice(releases, func(i, j int) bool {
			dateI, _ := time.Parse(models.DateFormat, releases[i].Date)
			dateJ, _ := time.Parse(models.DateFormat, releases[j].Date)
			return dateI.Before(dateJ)
		})

		var bestRelease models.Release
		found := false
		for _, release := range releases {
			if !found {
				bestRelease = release
				found = true
				continue
			}
			if release.TitleTrack != "N/A" && release.MV != "" {
				bestRelease = release
				break
			} else if release.TitleTrack != "N/A" && bestRelease.TitleTrack == "N/A" {
				bestRelease = release
			} else if release.MV != "" && bestRelease.MV == "" {
				bestRelease = release
			}
		}

		allReleases = append(allReleases, bestRelease)
	}

	sort.Slice(allReleases, func(i, j int) bool {
		dateI, _ := time.Parse(models.DateFormat, allReleases[i].Date)
		dateJ, _ := time.Parse(models.DateFormat, allReleases[j].Date)
		return dateI.Before(dateJ)
	})

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
		return formatter.CleanLink(lastLink, logger)
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
