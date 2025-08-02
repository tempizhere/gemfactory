package artist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// whitelistManagerImpl implements the WhitelistManager interface
type whitelistManagerImpl struct {
	female map[string]struct{}
	male   map[string]struct{}
	united []string
	dir    string
	logger *zap.Logger
	mu     sync.Mutex
	loaded bool
}

// NewWhitelistManager creates a new WhitelistManager instance
func NewWhitelistManager(dir string, logger *zap.Logger) WhitelistManager {
	manager := &whitelistManagerImpl{
		female: make(map[string]struct{}),
		male:   make(map[string]struct{}),
		dir:    dir,
		logger: logger,
	}
	
	// Попытка переноса старых файлов вайтлистов
	manager.migrateOldWhitelists()
	
	return manager
}

// loadWhitelists loads the female and male whitelists from JSON files
func (m *whitelistManagerImpl) loadWhitelists() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loaded {
		return nil
	}

	if err := m.loadFemaleWhitelist(); err != nil {
		return fmt.Errorf("failed to load female whitelist: %w", err)
	}
	if err := m.loadMaleWhitelist(); err != nil {
		return fmt.Errorf("failed to load male whitelist: %w", err)
	}
	m.updateUnited()
	m.loaded = true
	return nil
}

// loadFemaleWhitelist loads the female whitelist from JSON file
func (m *whitelistManagerImpl) loadFemaleWhitelist() error {
	path := filepath.Join(m.dir, "female_whitelist.json")
	cleanPath := filepath.Clean(path)

	// Базовая защита от path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid file path: %s", path)
	}

	// Создаем директорию, если она не существует
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		m.logger.Error("Failed to create directory", zap.String("dir", m.dir), zap.Error(err))
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует, создаем пустой файл
			m.logger.Info("Female whitelist file not found, creating empty file", zap.String("path", cleanPath))
			if err := m.saveWhitelist("female_whitelist.json", m.female); err != nil {
				m.logger.Error("Failed to create empty female whitelist", zap.Error(err))
				return fmt.Errorf("failed to create empty female whitelist: %w", err)
			}
			return nil
		}
		m.logger.Error("Failed to read female whitelist", zap.Error(err))
		return err
	}

	var items []string
	if err := json.Unmarshal(data, &items); err != nil {
		m.logger.Error("Failed to unmarshal female whitelist", zap.Error(err))
		return err
	}

	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		if key != "" {
			m.female[key] = struct{}{}
		}
	}
	return nil
}

// loadMaleWhitelist loads the male whitelist from JSON file
func (m *whitelistManagerImpl) loadMaleWhitelist() error {
	path := filepath.Join(m.dir, "male_whitelist.json")
	cleanPath := filepath.Clean(path)

	// Базовая защита от path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid file path: %s", path)
	}

	// Создаем директорию, если она не существует
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		m.logger.Error("Failed to create directory", zap.String("dir", m.dir), zap.Error(err))
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует, создаем пустой файл
			m.logger.Info("Male whitelist file not found, creating empty file", zap.String("path", cleanPath))
			if err := m.saveWhitelist("male_whitelist.json", m.male); err != nil {
				m.logger.Error("Failed to create empty male whitelist", zap.Error(err))
				return fmt.Errorf("failed to create empty male whitelist: %w", err)
			}
			return nil
		}
		m.logger.Error("Failed to read male whitelist", zap.Error(err))
		return err
	}

	var items []string
	if err := json.Unmarshal(data, &items); err != nil {
		m.logger.Error("Failed to unmarshal male whitelist", zap.Error(err))
		return err
	}

	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		if key != "" {
			m.male[key] = struct{}{}
		}
	}
	return nil
}

// updateUnited updates the united whitelist
func (m *whitelistManagerImpl) updateUnited() {
	m.united = nil
	for k := range m.female {
		m.united = append(m.united, k)
	}
	for k := range m.male {
		if _, exists := m.female[k]; !exists {
			m.united = append(m.united, k)
		}
	}
}

// GetFemaleWhitelist returns the female whitelist
func (m *whitelistManagerImpl) GetFemaleWhitelist() []string {
	if err := m.loadWhitelists(); err != nil {
		m.logger.Error("Failed to load whitelists", zap.Error(err))
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]string, 0, len(m.female))
	for k := range m.female {
		result = append(result, k)
	}
	return result
}

// GetMaleWhitelist returns the male whitelist
func (m *whitelistManagerImpl) GetMaleWhitelist() []string {
	if err := m.loadWhitelists(); err != nil {
		m.logger.Error("Failed to load whitelists", zap.Error(err))
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]string, 0, len(m.male))
	for k := range m.male {
		result = append(result, k)
	}
	return result
}

// GetUnitedWhitelist returns the united whitelist
func (m *whitelistManagerImpl) GetUnitedWhitelist() []string {
	if err := m.loadWhitelists(); err != nil {
		m.logger.Error("Failed to load whitelists", zap.Error(err))
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]string, len(m.united))
	copy(result, m.united)
	return result
}

// AddArtist adds an artist to the specified whitelist and saves to JSON
func (m *whitelistManagerImpl) AddArtist(artist string, isFemale bool) error {
	if err := m.loadWhitelists(); err != nil {
		return fmt.Errorf("failed to load whitelists: %w", err)
	}

	artist = strings.ToLower(strings.TrimSpace(artist))
	if artist == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if isFemale {
		m.female[artist] = struct{}{}
	} else {
		m.male[artist] = struct{}{}
	}
	m.updateUnited()

	var err error
	if isFemale {
		err = m.saveWhitelist("female_whitelist.json", m.female)
	} else {
		err = m.saveWhitelist("male_whitelist.json", m.male)
	}
	if err != nil {
		return fmt.Errorf("failed to save whitelist: %w", err)
	}
	m.logger.Info("Added artist", zap.String("artist", artist), zap.Bool("isFemale", isFemale))
	return nil
}

// AddArtists adds multiple artists to the specified whitelist and saves to JSON
func (m *whitelistManagerImpl) AddArtists(artists []string, isFemale bool) (int, error) {
	if err := m.loadWhitelists(); err != nil {
		return 0, fmt.Errorf("failed to load whitelists: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	addedCount := 0
	for _, artist := range artists {
		artist = strings.ToLower(strings.TrimSpace(artist))
		if artist == "" {
			continue
		}

		if isFemale {
			if _, exists := m.female[artist]; !exists {
				m.female[artist] = struct{}{}
				addedCount++
			}
		} else {
			if _, exists := m.male[artist]; !exists {
				m.male[artist] = struct{}{}
				addedCount++
			}
		}
	}

	if addedCount == 0 {
		return 0, nil
	}

	m.updateUnited()

	var err error
	if isFemale {
		err = m.saveWhitelist("female_whitelist.json", m.female)
	} else {
		err = m.saveWhitelist("male_whitelist.json", m.male)
	}
	if err != nil {
		return addedCount, fmt.Errorf("failed to save whitelist: %w", err)
	}
	m.logger.Info("Added artists", zap.Int("count", addedCount), zap.Bool("isFemale", isFemale))
	return addedCount, nil
}

// RemoveArtist removes an artist from both whitelists and saves to JSON
func (m *whitelistManagerImpl) RemoveArtist(artist string) error {
	if err := m.loadWhitelists(); err != nil {
		return fmt.Errorf("failed to load whitelists: %w", err)
	}

	artist = strings.ToLower(strings.TrimSpace(artist))
	if artist == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	deletedFemale := false
	deletedMale := false

	if _, exists := m.female[artist]; exists {
		delete(m.female, artist)
		deletedFemale = true
	}
	if _, exists := m.male[artist]; exists {
		delete(m.male, artist)
		deletedMale = true
	}
	if deletedFemale || deletedMale {
		m.updateUnited()
	}

	var err error
	if deletedFemale {
		err = m.saveWhitelist("female_whitelist.json", m.female)
	}
	if err == nil && deletedMale {
		err = m.saveWhitelist("male_whitelist.json", m.male)
	}
	if err != nil {
		return fmt.Errorf("failed to save whitelist: %w", err)
	}
	if deletedFemale || deletedMale {
		m.logger.Info("Removed artist", zap.String("artist", artist))
	}
	return nil
}

// RemoveArtists removes multiple artists from both whitelists and saves to JSON
func (m *whitelistManagerImpl) RemoveArtists(artists []string) (int, error) {
	if err := m.loadWhitelists(); err != nil {
		return 0, fmt.Errorf("failed to load whitelists: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	removedCount := 0
	for _, artist := range artists {
		artist = strings.ToLower(strings.TrimSpace(artist))
		if artist == "" {
			continue
		}

		if _, exists := m.female[artist]; exists {
			delete(m.female, artist)
			removedCount++
		}
		if _, exists := m.male[artist]; exists {
			delete(m.male, artist)
			removedCount++
		}
	}

	if removedCount == 0 {
		return 0, nil
	}

	m.updateUnited()

	var err error
	if err = m.saveWhitelist("female_whitelist.json", m.female); err != nil {
		return removedCount, fmt.Errorf("failed to save female whitelist: %w", err)
	}
	if err = m.saveWhitelist("male_whitelist.json", m.male); err != nil {
		return removedCount, fmt.Errorf("failed to save male whitelist: %w", err)
	}
	m.logger.Info("Removed artists", zap.Int("count", removedCount))
	return removedCount, nil
}

// ClearWhitelists clears both whitelists and saves to JSON
func (m *whitelistManagerImpl) ClearWhitelists() error {
	if err := m.loadWhitelists(); err != nil {
		return fmt.Errorf("failed to load whitelists: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.female = make(map[string]struct{})
	m.male = make(map[string]struct{})
	m.united = nil

	if err := m.saveWhitelist("female_whitelist.json", m.female); err != nil {
		return fmt.Errorf("failed to save female whitelist: %w", err)
	}
	if err := m.saveWhitelist("male_whitelist.json", m.male); err != nil {
		return fmt.Errorf("failed to save male whitelist: %w", err)
	}
	m.logger.Info("Cleared whitelists")
	return nil
}

// saveWhitelist saves the specified whitelist to JSON
func (m *whitelistManagerImpl) saveWhitelist(filename string, items map[string]struct{}) error {
	path := filepath.Join(m.dir, filename)
	itemList := make([]string, 0, len(items))
	for k := range items {
		itemList = append(itemList, k)
	}

	data, err := json.MarshalIndent(itemList, "", "  ")
	if err != nil {
		m.logger.Error("Failed to marshal whitelist", zap.String("filename", filename), zap.Error(err))
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		m.logger.Error("Failed to write whitelist", zap.String("filename", filename), zap.Error(err))
		return err
	}
	return nil
}

// migrateOldWhitelists attempts to migrate old whitelist files from the root directory
func (m *whitelistManagerImpl) migrateOldWhitelists() {
	// Проверяем, существуют ли файлы в новой директории
	newFemalePath := filepath.Join(m.dir, "female_whitelist.json")
	newMalePath := filepath.Join(m.dir, "male_whitelist.json")
	
	// Если файлы уже существуют в новой директории, не мигрируем
	if _, err := os.Stat(newFemalePath); err == nil {
		if _, err := os.Stat(newMalePath); err == nil {
			m.logger.Info("Whitelist files already exist in new directory, skipping migration")
			return
		}
	}
	
	// Старые пути (в корневой директории)
	oldFemalePath := "female_whitelist.json"
	oldMalePath := "male_whitelist.json"
	
	// Создаем новую директорию
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		m.logger.Error("Failed to create directory for migration", zap.Error(err))
		return
	}
	
	// Мигрируем женский вайтлист
	if _, err := os.Stat(oldFemalePath); err == nil {
		if err := m.migrateFile(oldFemalePath, newFemalePath); err != nil {
			m.logger.Error("Failed to migrate female whitelist", zap.Error(err))
		} else {
			m.logger.Info("Successfully migrated female whitelist", 
				zap.String("from", oldFemalePath), 
				zap.String("to", newFemalePath))
		}
	}
	
	// Мигрируем мужской вайтлист
	if _, err := os.Stat(oldMalePath); err == nil {
		if err := m.migrateFile(oldMalePath, newMalePath); err != nil {
			m.logger.Error("Failed to migrate male whitelist", zap.Error(err))
		} else {
			m.logger.Info("Successfully migrated male whitelist", 
				zap.String("from", oldMalePath), 
				zap.String("to", newMalePath))
		}
	}
}

// migrateFile copies a file from old path to new path
func (m *whitelistManagerImpl) migrateFile(oldPath, newPath string) error {
	// Читаем старый файл
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read old file: %w", err)
	}
	
	// Записываем в новый файл
	if err := os.WriteFile(newPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write new file: %w", err)
	}
	
	// Удаляем старый файл
	if err := os.Remove(oldPath); err != nil {
		m.logger.Warn("Failed to remove old file after migration", 
			zap.String("path", oldPath), zap.Error(err))
		// Не возвращаем ошибку, так как миграция прошла успешно
	}
	
	return nil
}
