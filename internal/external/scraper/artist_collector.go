package scraper

import (
	"regexp"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// collectArtistBlock собирает блок с артистом для LLM обработки
func (f *fetcherImpl) collectArtistBlock(blockHTML string, monthNum, year string, artists map[string]bool, artistBlocks *[]ArtistBlock, mu *sync.Mutex, rowCount int) {
	// Извлекаем артиста из блока - ищем <strong><mark> теги с артистами
	artistPattern := regexp.MustCompile(`<strong><mark[^>]*class="has-inline-color has-red-color"[^>]*>([^<]+)</mark></strong>`)
	artistMatches := artistPattern.FindAllStringSubmatch(blockHTML, -1)

	if len(artistMatches) == 0 {
		blockPreview := blockHTML
		if len(blockHTML) > 500 {
			blockPreview = blockHTML[:500]
		}
		f.logger.Debug("No artist found in block", zap.String("block", blockPreview))
		return
	}

	// Проверяем каждого найденного артиста
	for _, match := range artistMatches {
		if len(match) < 2 {
			continue
		}

		artist := strings.TrimSpace(match[1])
		artistKey := strings.ToLower(artist)

		// Проверяем, есть ли артист в списке для фильтрации
		if _, ok := artists[artistKey]; ok {
			f.logger.Info("Found active artist in block", zap.String("artist", artist), zap.Int("row", rowCount))

			// Добавляем блок в коллекцию
			mu.Lock()
			*artistBlocks = append(*artistBlocks, ArtistBlock{
				HTML:   blockHTML,
				Artist: artist,
				Row:    rowCount,
			})
			mu.Unlock()

			f.logger.Debug("Added artist block for LLM processing",
				zap.String("artist", artist),
				zap.Int("row", rowCount),
				zap.Int("total_blocks", len(*artistBlocks)))
			return // Нашли одного артиста из списка, этого достаточно
		}
	}

	// Если дошли сюда, значит ни один артист из блока не в списке
	firstArtist := ""
	if len(artistMatches) > 0 && len(artistMatches[0]) > 1 {
		firstArtist = strings.TrimSpace(artistMatches[0][1])
	}
	f.logger.Debug("Artist not in filter list", zap.String("artist", firstArtist), zap.Int("row", rowCount))
}
