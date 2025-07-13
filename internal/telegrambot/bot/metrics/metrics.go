// Package metrics реализует систему метрик для Telegram-бота.
package metrics

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Metrics представляет систему метрик бота
type Metrics struct {
	mu sync.RWMutex

	// Пользовательская активность
	totalCommands int64
	uniqueUsers   map[int64]struct{}

	// Метрики артистов
	femaleArtists int
	maleArtists   int
	totalArtists  int

	// Метрики релизов
	cachedReleases int
	cacheHitRate   float64
	cacheMisses    int64
	cacheHits      int64

	// Метрики производительности
	avgResponseTime time.Duration
	totalRequests   int64
	errorCount      int64

	// Системные метрики
	lastCacheUpdate time.Time
	nextCacheUpdate time.Time
	uptime          time.Time

	logger *zap.Logger
}

// NewMetrics создает новую систему метрик
func NewMetrics(logger *zap.Logger) *Metrics {
	return &Metrics{
		uniqueUsers: make(map[int64]struct{}),
		uptime:      time.Now(),
		logger:      logger,
	}
}

// RecordUserCommand записывает выполнение пользовательской команды
func (m *Metrics) RecordUserCommand(command string, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalCommands++
	m.uniqueUsers[userID] = struct{}{}
}

// UpdateArtistMetrics обновляет метрики артистов
func (m *Metrics) UpdateArtistMetrics(femaleCount, maleCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.femaleArtists = femaleCount
	m.maleArtists = maleCount
	m.totalArtists = femaleCount + maleCount
}

// UpdateReleaseMetrics обновляет метрики релизов
func (m *Metrics) UpdateReleaseMetrics(cachedCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cachedReleases = cachedCount
}

// RecordCacheHit записывает попадание в кэш
func (m *Metrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheHits++
	m.updateCacheHitRate()
}

// RecordCacheMiss записывает промах кэша
func (m *Metrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheMisses++
	m.updateCacheHitRate()
}

// RecordResponseTime записывает время ответа
func (m *Metrics) RecordResponseTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	// Простое скользящее среднее
	if m.avgResponseTime == 0 {
		m.avgResponseTime = duration
	} else {
		m.avgResponseTime = (m.avgResponseTime + duration) / 2
	}
}

// RecordError записывает ошибку
func (m *Metrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorCount++
}

// SetCacheUpdateStatus устанавливает статус обновления кэша
func (m *Metrics) SetCacheUpdateStatus(updating bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !updating {
		m.lastCacheUpdate = time.Now()
	}
}

// SetNextCacheUpdate устанавливает время следующего обновления кэша
func (m *Metrics) SetNextCacheUpdate(nextUpdate time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextCacheUpdate = nextUpdate
}

// GetStats возвращает все метрики в виде map
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"user_activity": map[string]interface{}{
			"total_commands": m.totalCommands,
			"unique_users":   len(m.uniqueUsers),
		},
		"artists": map[string]interface{}{
			"female_artists": m.femaleArtists,
			"male_artists":   m.maleArtists,
			"total_artists":  m.totalArtists,
		},
		"releases": map[string]interface{}{
			"cached_releases": m.cachedReleases,
			"cache_hit_rate":  m.cacheHitRate,
			"cache_hits":      m.cacheHits,
			"cache_misses":    m.cacheMisses,
		},
		"performance": map[string]interface{}{
			"avg_response_time": m.formatDuration(m.avgResponseTime),
			"total_requests":    m.totalRequests,
			"error_count":       m.errorCount,
			"error_rate":        m.calculateErrorRate(),
		},
		"system": map[string]interface{}{
			"uptime":            m.formatDuration(time.Since(m.uptime)),
			"last_cache_update": m.formatTime(m.lastCacheUpdate),
			"next_cache_update": m.formatTime(m.nextCacheUpdate),
		},
	}
}

// updateCacheHitRate обновляет процент попаданий в кэш
func (m *Metrics) updateCacheHitRate() {
	total := m.cacheHits + m.cacheMisses
	if total > 0 {
		m.cacheHitRate = float64(m.cacheHits) / float64(total) * 100
	}
}

// calculateErrorRate вычисляет процент ошибок
func (m *Metrics) calculateErrorRate() float64 {
	if m.totalRequests > 0 {
		return float64(m.errorCount) / float64(m.totalRequests) * 100
	}
	return 0
}

// formatTime форматирует время в нужном формате или возвращает "Не установлено"
func (m *Metrics) formatTime(t time.Time) string {
	if t.IsZero() {
		return "Не установлено"
	}
	return t.Format("02.01.06 15:04")
}

// formatDuration форматирует duration с двумя знаками после запятой
func (m *Metrics) formatDuration(d time.Duration) string {
	// Конвертируем в секунды с двумя знаками после запятой
	seconds := d.Seconds()
	return fmt.Sprintf("%.2fs", seconds)
}
