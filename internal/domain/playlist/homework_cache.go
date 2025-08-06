// Package playlist содержит кэш для отслеживания запросов домашних заданий.
package playlist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gemfactory/internal/domain/types"
	"gemfactory/internal/gateway/spotify"

	"go.uber.org/zap"
)

// HomeworkInfo содержит информацию о выданном домашнем задании
type HomeworkInfo struct {
	RequestTime time.Time
	Track       *spotify.Track
	PlayCount   int
}

// HomeworkCache кэширует запросы домашних заданий пользователей
type HomeworkCache struct {
	requests    map[int64]*HomeworkInfo // userID -> homework info
	mu          sync.RWMutex
	location    *time.Location // Временная зона для расчетов
	storagePath string         // Путь для сохранения кэша
	logger      *zap.Logger
}

// NewHomeworkCache создает новый кэш домашних заданий
func NewHomeworkCache() *HomeworkCache {
	return &HomeworkCache{
		requests: make(map[int64]*HomeworkInfo),
		location: time.UTC, // По умолчанию UTC
	}
}

// SetStoragePath устанавливает путь для сохранения кэша
func (c *HomeworkCache) SetStoragePath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storagePath = path
}

// SetLogger устанавливает логгер для кэша
func (c *HomeworkCache) SetLogger(logger *zap.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger = logger
}

// LoadFromStorage загружает кэш из файла
func (c *HomeworkCache) LoadFromStorage() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.storagePath == "" {
		return nil // Нет пути для сохранения
	}

	// Создаем директорию если не существует
	dir := filepath.Dir(c.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to create storage directory", zap.Error(err))
		}
		return err
	}

	// Проверяем существование файла
	if _, err := os.Stat(c.storagePath); os.IsNotExist(err) {
		if c.logger != nil {
			c.logger.Info("Homework cache file does not exist, starting with empty cache")
		}
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(c.storagePath)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to read homework cache file", zap.Error(err))
		}
		return err
	}

	// Десериализуем данные
	var cacheData map[int64]*HomeworkInfo
	if err := json.Unmarshal(data, &cacheData); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to unmarshal homework cache data", zap.Error(err))
		}
		return err
	}

	c.requests = cacheData
	if c.logger != nil {
		c.logger.Info("Homework cache loaded from storage", zap.Int("entries", len(c.requests)))
	}

	return nil
}

// SaveToStorage сохраняет кэш в файл
func (c *HomeworkCache) SaveToStorage() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.storagePath == "" {
		return nil // Нет пути для сохранения
	}

	// Создаем директорию если не существует
	dir := filepath.Dir(c.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to create storage directory", zap.Error(err))
		}
		return err
	}

	// Сериализуем данные
	data, err := json.MarshalIndent(c.requests, "", "  ")
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to marshal homework cache data", zap.Error(err))
		}
		return err
	}

	// Записываем в файл
	if err := os.WriteFile(c.storagePath, data, 0644); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to write homework cache file", zap.Error(err))
		}
		return err
	}

	if c.logger != nil {
		c.logger.Debug("Homework cache saved to storage", zap.Int("entries", len(c.requests)))
	}

	return nil
}

// SetLocation устанавливает временную зону для кэша
func (c *HomeworkCache) SetLocation(location *time.Location) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Просто устанавливаем новую временную зону без очистки кэша
	// Существующие записи будут корректно работать с новой временной зоной
	c.location = location
}

// CanRequest проверяет, может ли пользователь запросить домашнее задание
func (c *HomeworkCache) CanRequest(userID int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	homeworkInfo, exists := c.requests[userID]
	if !exists {
		if c.logger != nil {
			c.logger.Debug("User has no previous homework request", zap.Int64("user_id", userID))
		}
		return true // Первый запрос
	}

	// Проверяем, наступила ли полночь с момента последнего запроса
	now := time.Now().In(c.location)

	// Получаем компоненты даты последнего запроса
	lastYear, lastMonth, lastDay := homeworkInfo.RequestTime.In(c.location).Date()

	// Получаем компоненты текущей даты
	currentYear, currentMonth, currentDay := now.Date()

	// Сравниваем даты: если текущая дата больше даты последнего запроса, то можно запросить
	canRequest := currentYear > lastYear ||
		(currentYear == lastYear && currentMonth > lastMonth) ||
		(currentYear == lastYear && currentMonth == lastMonth && currentDay > lastDay)

	if c.logger != nil {
		c.logger.Debug("Homework request check",
			zap.Int64("user_id", userID),
			zap.Time("last_request", homeworkInfo.RequestTime),
			zap.Int("last_year", lastYear),
			zap.String("last_month", lastMonth.String()),
			zap.Int("last_day", lastDay),
			zap.Int("current_year", currentYear),
			zap.String("current_month", currentMonth.String()),
			zap.Int("current_day", currentDay),
			zap.Bool("can_request", canRequest),
			zap.String("timezone", c.location.String()))
	}

	return canRequest
}

// RecordRequest записывает запрос пользователя
func (c *HomeworkCache) RecordRequest(userID int64, track *spotify.Track, playCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	requestTime := time.Now().In(c.location)
	c.requests[userID] = &HomeworkInfo{
		RequestTime: requestTime,
		Track:       track,
		PlayCount:   playCount,
	}

	if c.logger != nil {
		c.logger.Info("Homework request recorded",
			zap.Int64("user_id", userID),
			zap.String("track_id", track.ID),
			zap.String("track_title", track.Title),
			zap.String("track_artist", track.Artist),
			zap.Int("play_count", playCount),
			zap.Time("request_time", requestTime))
	}

	// Сохраняем кэш в фоновом режиме
	go func() {
		if err := c.SaveToStorage(); err != nil {
			if c.logger != nil {
				c.logger.Error("Failed to save homework cache", zap.Error(err))
			}
		}
	}()
}

// GetTimeUntilNextRequest возвращает время до следующего возможного запроса (до полуночи)
func (c *HomeworkCache) GetTimeUntilNextRequest(userID int64) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	homeworkInfo, exists := c.requests[userID]
	if !exists {
		return 0 // Можно запросить сразу
	}

	now := time.Now().In(c.location)

	// Получаем компоненты даты последнего запроса
	lastYear, lastMonth, lastDay := homeworkInfo.RequestTime.In(c.location).Date()

	// Получаем компоненты текущей даты
	currentYear, currentMonth, currentDay := now.Date()

	// Если текущая дата больше даты последнего запроса, можно запросить
	if currentYear > lastYear ||
		(currentYear == lastYear && currentMonth > lastMonth) ||
		(currentYear == lastYear && currentMonth == lastMonth && currentDay > lastDay) {
		return 0
	}

	// Вычисляем время до ближайшей полуночи от текущего времени
	// Создаем время полуночи в той же временной зоне
	year, month, day := now.Date()
	nextMidnight := time.Date(year, month, day+1, 0, 0, 0, 0, c.location)
	duration := nextMidnight.Sub(now)

	if c.logger != nil {
		c.logger.Debug("Time until next request calculation",
			zap.Int64("user_id", userID),
			zap.Time("now", now),
			zap.Int("last_year", lastYear),
			zap.String("last_month", lastMonth.String()),
			zap.Int("last_day", lastDay),
			zap.Int("current_year", currentYear),
			zap.String("current_month", currentMonth.String()),
			zap.Int("current_day", currentDay),
			zap.Time("next_midnight", nextMidnight),
			zap.Duration("duration", duration))
	}

	return duration
}

// Cleanup удаляет старые записи (старше 48 часов)
func (c *HomeworkCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().In(c.location).Add(-48 * time.Hour)
	for userID, homeworkInfo := range c.requests {
		if homeworkInfo.RequestTime.Before(cutoff) {
			delete(c.requests, userID)
		}
	}
}

// GetTotalRequests возвращает общее количество запросов
func (c *HomeworkCache) GetTotalRequests() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.requests)
}

// GetUniqueUsers возвращает количество уникальных пользователей
func (c *HomeworkCache) GetUniqueUsers() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.requests)
}

// GetHomeworkInfo возвращает информацию о домашнем задании пользователя
func (c *HomeworkCache) GetHomeworkInfo(userID int64) *types.HomeworkInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	homeworkInfo, exists := c.requests[userID]
	if !exists {
		return nil
	}

	return &types.HomeworkInfo{
		RequestTime: homeworkInfo.RequestTime,
		Track:       homeworkInfo.Track,
		PlayCount:   homeworkInfo.PlayCount,
	}
}
