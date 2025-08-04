// Package playlist содержит сервисы для работы с плейлистами.
package playlist

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"sync"

	"go.uber.org/zap"
)

// playlistServiceImpl реализует интерфейс PlaylistService
type playlistServiceImpl struct {
	tracks []*Track
	mu     sync.RWMutex
	loaded bool
	logger *zap.Logger
}

// NewPlaylistService создает новый экземпляр PlaylistService
func NewPlaylistService(logger *zap.Logger) PlaylistService {
	return &playlistServiceImpl{
		tracks: make([]*Track, 0),
		logger: logger,
	}
}

// LoadPlaylist загружает плейлист из CSV файла
func (p *playlistServiceImpl) LoadPlaylist(filePath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Loading playlist from CSV file", zap.String("file_path", filePath))

	file, err := os.Open(filePath)
	if err != nil {
		p.logger.Error("Failed to open CSV file", zap.String("file_path", filePath), zap.Error(err))
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			p.logger.Error("Failed to close file", zap.Error(err))
		}
	}()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		p.logger.Error("Failed to read CSV file", zap.String("file_path", filePath), zap.Error(err))
		return fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) < 2 {
		p.logger.Error("CSV file is empty or contains only header", zap.String("file_path", filePath))
		return fmt.Errorf("CSV file is empty or contains only header")
	}

	// Пропускаем заголовок (первую строку)
	tracks := make([]*Track, 0, len(records)-1)

	for i, record := range records[1:] {
		if len(record) < 22 {
			p.logger.Warn("Skipping invalid record", zap.Int("row", i+2), zap.Int("columns", len(record)))
			continue
		}

		// Колонки: 0=#, 1=Song, 2=Artist, 20=Spotify Track Id
		track := &Track{
			ID:     record[20], // Spotify Track Id
			Title:  record[1],  // Song
			Artist: record[2],  // Artist
		}

		// Валидация данных
		if track.Title == "" || track.Artist == "" {
			p.logger.Warn("Skipping track with empty title or artist",
				zap.Int("row", i+2),
				zap.String("title", track.Title),
				zap.String("artist", track.Artist))
			continue
		}

		tracks = append(tracks, track)
	}

	p.tracks = tracks
	p.loaded = true

	p.logger.Info("Playlist loaded successfully",
		zap.Int("total_tracks", len(p.tracks)),
		zap.String("file_path", filePath))

	return nil
}

// GetRandomTrack возвращает случайный трек из плейлиста
func (p *playlistServiceImpl) GetRandomTrack() (*Track, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.loaded {
		return nil, fmt.Errorf("playlist not loaded")
	}

	if len(p.tracks) == 0 {
		return nil, fmt.Errorf("playlist is empty")
	}

	// Генерируем случайный индекс
	index := rand.Intn(len(p.tracks))

	track := p.tracks[index]
	p.logger.Debug("Selected random track",
		zap.String("artist", track.Artist),
		zap.String("title", track.Title),
		zap.Int("index", index))

	return track, nil
}

// GetTotalTracks возвращает общее количество треков в плейлисте
func (p *playlistServiceImpl) GetTotalTracks() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.tracks)
}

// IsLoaded проверяет, загружен ли плейлист
func (p *playlistServiceImpl) IsLoaded() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.loaded
}
