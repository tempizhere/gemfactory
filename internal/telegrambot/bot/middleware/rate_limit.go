package middleware

import (
	"gemfactory/internal/telegrambot/bot/types"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiter представляет rate limiter для пользователей
type RateLimiter struct {
	requests map[int64][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
	logger   *zap.Logger
}

// Убеждаемся, что RateLimiter реализует RateLimiterInterface
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

// RateLimit middleware ограничивает частоту запросов от пользователей
func (rl *RateLimiter) RateLimit(next types.HandlerFunc) types.HandlerFunc {
	return func(ctx types.Context) error {
		userID := ctx.Message.From.ID

		if !rl.allowRequest(userID) {
			rl.logger.Warn("Rate limit exceeded",
				zap.Int64("user_id", userID),
				zap.String("command", ctx.Message.Command()))

			return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID,
				"Слишком много запросов. Попробуйте позже.")
		}

		return next(ctx)
	}
}

// AllowRequest проверяет, разрешен ли запрос (публичный метод)
func (rl *RateLimiter) AllowRequest(userID int64) bool {
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

	// Удаляем старые запросы
	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Проверяем лимит
	if len(validRequests) >= rl.limit {
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
