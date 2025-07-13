package scraper

import (
	"gemfactory/internal/domain/service"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// ExtractYouTubeLinkFromEvent extracts YouTube link from an event
func ExtractYouTubeLinkFromEvent(e *colly.HTMLElement, startIndex, endIndex int, logger *zap.Logger) string {
	var lastLink string
	var currentIndex int
	var allLinks []string

	if startIndex < 0 {
		startIndex = 0
	}

	e.ForEach("td.has-text-align-left", func(_ int, s *colly.HTMLElement) {
		s.DOM.Contents().Each(func(_ int, node *goquery.Selection) {
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
		return service.CleanLink(lastLink, logger)
	}
	return ""
}

// ExtractAlbumName extracts album name from lines
func ExtractAlbumName(lines []string, startIndex, endIndex int, _ *zap.Logger) string {
	for i := startIndex; i < endIndex && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		lowerLine := strings.ToLower(line)

		switch {
		case strings.HasPrefix(lowerLine, "album:"):
			return strings.TrimSpace(strings.TrimPrefix(line, "album:"))
		case strings.HasPrefix(lowerLine, "ost:"):
			return strings.TrimSpace(strings.TrimPrefix(line, "ost:"))
		case strings.Contains(lowerLine, "mini album") || strings.Contains(lowerLine, "special mini album"):
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "N/A"
}

// ExtractTrackName extracts track name from lines
func ExtractTrackName(lines []string, startIndex, endIndex int, _ *zap.Logger) string {
	for i := startIndex; i < endIndex && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		line = strings.ReplaceAll(line, "‘", "'")
		line = strings.ReplaceAll(line, "’", "'")
		line = strings.ReplaceAll(line, "“", "\"")
		line = strings.ReplaceAll(line, "”", "\"")

		lowerLine := strings.ToLower(line)
		var trackName string

		switch {
		case strings.HasPrefix(lowerLine, "title track:"):
			trackName = strings.TrimSpace(strings.TrimPrefix(line, "title track:"))
		case strings.Contains(lowerLine, "release") || strings.Contains(lowerLine, "pre-release") || strings.Contains(lowerLine, "mv release"):
			trackName = line
		default:
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
