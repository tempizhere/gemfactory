// Package cache реализует кэширование команд для Telegram-бота.
package cache

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// CommandCache представляет кэш для результатов команд
type CommandCache struct {
	cache  map[string]Entry
	mu     sync.RWMutex
	ttl    time.Duration
	logger *zap.Logger
}

// Убеждаемся, что CommandCache реализует CommandCacheInterface
var _ CommandCacheInterface = (*CommandCache)(nil)

// Entry представляет запись в кэше
type Entry struct {
	Data      any
	Timestamp time.Time
}

// NewCommandCache создает новый кэш команд
func NewCommandCache(ttl time.Duration, logger *zap.Logger) *CommandCache {
	cc := &CommandCache{
		cache:  make(map[string]Entry),
		ttl:    ttl,
		logger: logger,
	}

	// Запускаем очистку устаревших записей
	go cc.cleanupLoop()

	return cc
}

// Get получает значение из кэша
func (cc *CommandCache) Get(key string) (any, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	entry, exists := cc.cache[key]
	if !exists {
		return nil, false
	}

	// Проверяем TTL
	if time.Since(entry.Timestamp) > cc.ttl {
		// Удаляем устаревшую запись
		cc.mu.RUnlock()
		cc.mu.Lock()
		delete(cc.cache, key)
		cc.mu.Unlock()
		cc.mu.RLock()
		return nil, false
	}

	cc.logger.Debug("Cache hit", zap.String("key", key))
	return entry.Data, true
}

// Set устанавливает значение в кэш
func (cc *CommandCache) Set(key string, data any) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.cache[key] = Entry{
		Data:      data,
		Timestamp: time.Now(),
	}

	cc.logger.Debug("Cache set", zap.String("key", key))
}

// Delete удаляет значение из кэша
func (cc *CommandCache) Delete(key string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	delete(cc.cache, key)
	cc.logger.Debug("Cache delete", zap.String("key", key))
}

// Clear очищает весь кэш
func (cc *CommandCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.cache = make(map[string]Entry)
	cc.logger.Info("Cache cleared")
}

// cleanupLoop периодически очищает устаревшие записи
func (cc *CommandCache) cleanupLoop() {
	ticker := time.NewTicker(cc.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		cc.cleanup()
	}
}

// cleanup очищает устаревшие записи
func (cc *CommandCache) cleanup() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	now := time.Now()
	deleted := 0

	for key, entry := range cc.cache {
		if now.Sub(entry.Timestamp) > cc.ttl {
			delete(cc.cache, key)
			deleted++
		}
	}

	if deleted > 0 {
		cc.logger.Debug("Cleaned up cache entries", zap.Int("deleted", deleted))
	}
}

// Stats возвращает статистику кэша
func (cc *CommandCache) Stats() map[string]any {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return map[string]any{
		"size": len(cc.cache),
		"ttl":  cc.ttl.String(),
	}
}
