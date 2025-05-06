package parser

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"

	"gemfactory/internal/telegrambot/releases/releasefmt"
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
