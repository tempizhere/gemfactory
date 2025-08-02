// Package playlist содержит кэш для отслеживания запросов домашних заданий.
package playlist

import (
	"sync"
	"time"
)

// HomeworkCache кэширует запросы домашних заданий пользователей
type HomeworkCache struct {
	requests map[int64]time.Time // userID -> last request time
	mu       sync.RWMutex
}

// NewHomeworkCache создает новый кэш домашних заданий
func NewHomeworkCache() *HomeworkCache {
	return &HomeworkCache{
		requests: make(map[int64]time.Time),
	}
}

// CanRequest проверяет, может ли пользователь запросить домашнее задание
func (c *HomeworkCache) CanRequest(userID int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	lastRequest, exists := c.requests[userID]
	if !exists {
		return true // Первый запрос
	}

	// Проверяем, прошло ли 24 часа с последнего запроса
	return time.Since(lastRequest) >= 24*time.Hour
}

// RecordRequest записывает запрос пользователя
func (c *HomeworkCache) RecordRequest(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests[userID] = time.Now()
}

// GetTimeUntilNextRequest возвращает время до следующего возможного запроса
func (c *HomeworkCache) GetTimeUntilNextRequest(userID int64) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	lastRequest, exists := c.requests[userID]
	if !exists {
		return 0 // Можно запросить сразу
	}

	timeSinceLastRequest := time.Since(lastRequest)
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
	for userID, lastRequest := range c.requests {
		if lastRequest.Before(cutoff) {
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
