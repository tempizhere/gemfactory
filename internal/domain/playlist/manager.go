// Package playlist содержит менеджер для работы с плейлистами.
package playlist

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

// Manager управляет плейлистами
type Manager struct {
	tracks     []*Track
	mu         sync.RWMutex
	loaded     bool
	logger     *zap.Logger
	storageDir string
}

var _ PlaylistManager = (*Manager)(nil)

// NewManager создает новый менеджер плейлистов
func NewManager(logger *zap.Logger, storageDir string) *Manager {
	return &Manager{
		tracks:     make([]*Track, 0),
		logger:     logger,
		storageDir: storageDir,
	}
}

// LoadPlaylistFromFile загружает плейлист из файла
func (m *Manager) LoadPlaylistFromFile(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Loading playlist from file", zap.String("file_path", filePath))

	file, err := os.Open(filePath)
	if err != nil {
		m.logger.Error("Failed to open playlist file", zap.String("file_path", filePath), zap.Error(err))
		return fmt.Errorf("failed to open playlist file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			m.logger.Error("Failed to close file", zap.Error(err))
		}
	}()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		m.logger.Error("Failed to read playlist file", zap.String("file_path", filePath), zap.Error(err))
		return fmt.Errorf("failed to read playlist file: %w", err)
	}

	if len(records) < 2 {
		m.logger.Error("Playlist file is empty or contains only header", zap.String("file_path", filePath))
		return fmt.Errorf("playlist file is empty or contains only header")
	}

	// Пропускаем заголовок (первую строку)
	tracks := make([]*Track, 0, len(records)-1)

	for i, record := range records[1:] {
		if len(record) < 22 {
			m.logger.Warn("Skipping invalid record", zap.Int("row", i+2), zap.Int("columns", len(record)))
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
			m.logger.Warn("Skipping track with empty title or artist",
				zap.Int("row", i+2),
				zap.String("title", track.Title),
				zap.String("artist", track.Artist))
			continue
		}

		tracks = append(tracks, track)
	}

	m.tracks = tracks
	m.loaded = true

	// Автоматически сохраняем в постоянное хранилище
	if err := m.SavePlaylistToStorage(); err != nil {
		m.logger.Warn("Failed to save playlist to storage", zap.Error(err))
		// Не возвращаем ошибку, так как загрузка прошла успешно
	}

	m.logger.Info("Playlist loaded successfully",
		zap.Int("total_tracks", len(m.tracks)),
		zap.String("file_path", filePath))

	return nil
}

// GetRandomTrack возвращает случайный трек из плейлиста
func (m *Manager) GetRandomTrack() (*Track, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return nil, fmt.Errorf("playlist not loaded")
	}

	if len(m.tracks) == 0 {
		return nil, fmt.Errorf("playlist is empty")
	}

	// Генерируем случайный индекс
	index := m.randomInt(len(m.tracks))

	track := m.tracks[index]
	m.logger.Debug("Selected random track",
		zap.String("artist", track.Artist),
		zap.String("title", track.Title),
		zap.Int("index", index))

	return track, nil
}

// GetTotalTracks возвращает общее количество треков в плейлисте
func (m *Manager) GetTotalTracks() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tracks)
}

// IsLoaded проверяет, загружен ли плейлист
func (m *Manager) IsLoaded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loaded
}

// Clear очищает плейлист
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tracks = make([]*Track, 0)
	m.loaded = false
	m.logger.Info("Playlist cleared")
}

// SavePlaylistToFile сохраняет плейлист в файл
func (m *Manager) SavePlaylistToFile(filePath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded || len(m.tracks) == 0 {
		return fmt.Errorf("no tracks to save")
	}

	// Создаем директорию, если она не существует
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.logger.Error("Failed to create directory", zap.String("dir", dir), zap.Error(err))
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		m.logger.Error("Failed to create playlist file", zap.String("file_path", filePath), zap.Error(err))
		return fmt.Errorf("failed to create playlist file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			m.logger.Error("Failed to close file", zap.Error(err))
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Записываем заголовок
	header := []string{"#", "Song", "Artist", "Spotify Track Id"}
	if err := writer.Write(header); err != nil {
		m.logger.Error("Failed to write header", zap.Error(err))
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Записываем треки
	for i, track := range m.tracks {
		record := []string{
			fmt.Sprintf("%d", i+1),
			track.Title,
			track.Artist,
			track.ID,
		}
		if err := writer.Write(record); err != nil {
			m.logger.Error("Failed to write track", zap.Int("index", i), zap.Error(err))
			return fmt.Errorf("failed to write track: %w", err)
		}
	}

	m.logger.Info("Playlist saved successfully", zap.String("file_path", filePath), zap.Int("tracks", len(m.tracks)))
	return nil
}

// LoadPlaylistFromStorage загружает плейлист из постоянного хранилища
func (m *Manager) LoadPlaylistFromStorage() error {
	storagePath := filepath.Join(m.storageDir, "playlist.csv")

	// Проверяем, существует ли файл
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		m.logger.Info("Playlist file not found in storage, will be loaded via /import_playlist command")
		return nil // Не возвращаем ошибку, если файл не существует
	}

	return m.LoadPlaylistFromFile(storagePath)
}

// SavePlaylistToStorage сохраняет плейлист в постоянное хранилище
func (m *Manager) SavePlaylistToStorage() error {
	// Создаем директорию, если она не существует
	if err := os.MkdirAll(m.storageDir, 0755); err != nil {
		m.logger.Error("Failed to create storage directory", zap.String("dir", m.storageDir), zap.Error(err))
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	storagePath := filepath.Join(m.storageDir, "playlist.csv")
	return m.SavePlaylistToFile(storagePath)
}

// randomInt генерирует случайное число
func (m *Manager) randomInt(n int) int {
	return rand.Intn(n)
}
