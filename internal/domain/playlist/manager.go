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
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open playlist file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty or has no data records")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Очищаем существующие треки
	m.tracks = make([]*Track, 0)

	// Пропускаем заголовок и обрабатываем данные
	for _, record := range records[1:] {
		if len(record) < 22 {
			continue
		}

		track := &Track{
			ID:     record[20], // Spotify Track Id
			Title:  record[1],  // Song
			Artist: record[2],  // Artist
		}

		// Проверяем, что у трека есть все необходимые данные
		if track.Title == "" || track.Artist == "" {
			continue
		}

		m.tracks = append(m.tracks, track)
	}

	m.loaded = true
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
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
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
	storagePath := filepath.Join(m.storageDir, "playlist.csv")
	return m.SavePlaylistToFile(storagePath)
}

// randomInt генерирует случайное число
func (m *Manager) randomInt(n int) int {
	return rand.Intn(n)
}
