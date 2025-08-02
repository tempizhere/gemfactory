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
}

// NewHomeworkCache создает новый кэш домашних заданий
func NewHomeworkCache() *HomeworkCache {
	return &HomeworkCache{
		requests: make(map[int64]*HomeworkInfo),
	}
}

// CanRequest проверяет, может ли пользователь запросить домашнее задание
func (c *HomeworkCache) CanRequest(userID int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	homeworkInfo, exists := c.requests[userID]
	if !exists {
		return true // Первый запрос
	}

	// Проверяем, прошло ли 24 часа с последнего запроса
	return time.Since(homeworkInfo.RequestTime) >= 24*time.Hour
}

// RecordRequest записывает запрос пользователя
func (c *HomeworkCache) RecordRequest(userID int64, track *Track, playCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests[userID] = &HomeworkInfo{
		RequestTime: time.Now(),
		Track:       track,
		PlayCount:   playCount,
	}
}

// GetTimeUntilNextRequest возвращает время до следующего возможного запроса
func (c *HomeworkCache) GetTimeUntilNextRequest(userID int64) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	homeworkInfo, exists := c.requests[userID]
	if !exists {
		return 0 // Можно запросить сразу
	}

	timeSinceLastRequest := time.Since(homeworkInfo.RequestTime)
	timeUntilNextRequest := 24*time.Hour - timeSinceLastRequest

	if timeUntilNextRequest <= 0 {
		return 0
	}

	return timeUntilNextRequest
}

// Cleanup удаляет старые записи (старше 48 часов)
func (c *HomeworkCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-48 * time.Hour)
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
