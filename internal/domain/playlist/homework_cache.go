// Package playlist содержит кэш для отслеживания запросов домашних заданий.
package playlist

import (
	"sync"
	"time"
)

// HomeworkInfo содержит информацию о выданном домашнем задании
type HomeworkInfo struct {
	RequestTime time.Time
	Track       *Track
	PlayCount   int
}

// HomeworkCache кэширует запросы домашних заданий пользователей
type HomeworkCache struct {
	requests map[int64]*HomeworkInfo // userID -> homework info
	mu       sync.RWMutex
	location *time.Location // Временная зона для расчетов
}

// NewHomeworkCache создает новый кэш домашних заданий
func NewHomeworkCache() *HomeworkCache {
	return &HomeworkCache{
		requests: make(map[int64]*HomeworkInfo),
		location: time.UTC, // По умолчанию UTC
	}
}

// SetLocation устанавливает временную зону для кэша
func (c *HomeworkCache) SetLocation(location *time.Location) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.location = location
}

// CanRequest проверяет, может ли пользователь запросить домашнее задание
func (c *HomeworkCache) CanRequest(userID int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	homeworkInfo, exists := c.requests[userID]
	if !exists {
		return true // Первый запрос
	}

	// Проверяем, наступила ли полночь с момента последнего запроса
	now := time.Now().In(c.location)
	lastRequestDate := homeworkInfo.RequestTime.In(c.location).Truncate(24 * time.Hour)
	currentDate := now.Truncate(24 * time.Hour)

	return currentDate.After(lastRequestDate)
}

// RecordRequest записывает запрос пользователя
func (c *HomeworkCache) RecordRequest(userID int64, track *Track, playCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests[userID] = &HomeworkInfo{
		RequestTime: time.Now().In(c.location),
		Track:       track,
		PlayCount:   playCount,
	}
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
	lastRequestDate := homeworkInfo.RequestTime.In(c.location).Truncate(24 * time.Hour)
	currentDate := now.Truncate(24 * time.Hour)

	// Если уже новый день, можно запросить
	if currentDate.After(lastRequestDate) {
		return 0
	}

	// Вычисляем время до следующей полуночи
	nextMidnight := currentDate.Add(24 * time.Hour)
	return nextMidnight.Sub(now)
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
func (c *HomeworkCache) GetHomeworkInfo(userID int64) *HomeworkInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	homeworkInfo, exists := c.requests[userID]
	if !exists {
		return nil
	}

	return homeworkInfo
}
