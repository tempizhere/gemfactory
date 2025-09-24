// Package middleware содержит middleware для rate limiting.
package middleware

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiterInterface определяет интерфейс для ограничителя запросов
type RateLimiterInterface interface {
	// AllowRequest проверяет, можно ли обработать запрос
	AllowRequest(userID int64) bool
	// Allow проверяет, разрешен ли запрос (для совместимости)
	Allow(userID int64) bool
	// Cleanup очищает устаревшие записи
	Cleanup()
}

// RateLimiter ограничивает количество запросов
type RateLimiter struct {
	requests map[int64][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
	logger   *zap.Logger
}

var _ RateLimiterInterface = (*RateLimiter)(nil)

// NewRateLimiter создает новый rate limiter
func NewRateLimiter(limit int, window time.Duration, logger *zap.Logger) *RateLimiter {
	return &RateLimiter{
		requests: make(map[int64][]time.Time),
		limit:    limit,
		window:   window,
		logger:   logger,
	}
}

// AllowRequest проверяет, разрешен ли запрос (публичный метод)
func (rl *RateLimiter) AllowRequest(userID int64) bool {
	return rl.allowRequest(userID)
}

// Allow проверяет, разрешен ли запрос (для совместимости)
func (rl *RateLimiter) Allow(userID int64) bool {
	return rl.allowRequest(userID)
}

// allowRequest проверяет, разрешен ли запрос
func (rl *RateLimiter) allowRequest(userID int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Получаем список запросов пользователя
	requests, exists := rl.requests[userID]
	if !exists {
		rl.requests[userID] = []time.Time{now}
		return true
	}

	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Проверяем лимит
	if len(validRequests) >= rl.limit {
		rl.logger.Warn("Rate limit exceeded",
			zap.Int64("user_id", userID),
			zap.Int("requests", len(validRequests)),
			zap.Int("limit", rl.limit))
		return false
	}

	// Добавляем новый запрос
	validRequests = append(validRequests, now)
	rl.requests[userID] = validRequests

	return true
}

// Cleanup очищает старые записи
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	for userID, requests := range rl.requests {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validRequests = append(validRequests, reqTime)
			}
		}

		if len(validRequests) == 0 {
			delete(rl.requests, userID)
		} else {
			rl.requests[userID] = validRequests
		}
	}
}
