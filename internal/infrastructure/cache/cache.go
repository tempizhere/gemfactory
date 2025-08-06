// Package cache реализует кэширование релизов для Telegram-бота.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gemfactory/internal/domain/release"
	"gemfactory/internal/infrastructure/worker"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// GetReleasesForMonths retrieves releases for multiple months from the cache
func (cm *Manager) GetReleasesForMonths(months []string, whitelist map[string]struct{}, _, _ bool) ([]release.Release, []string, error) {
	if len(whitelist) == 0 {
		cm.logger.Error("Whitelist is empty")
		return nil, nil, fmt.Errorf("whitelist is empty, please add artists using /add_artist")
	}

	var allReleases []release.Release
	var missingMonths []string
	cacheHit := false

	cm.mu.Lock()
	for _, month := range months {
		// Use united whitelist for cache key to match StoreReleases
		unitedWhitelist := cm.artistList.GetUnitedWhitelist()
		cacheKey := fmt.Sprintf("%s-%s", strings.ToLower(month), cm.HashWhitelist(unitedWhitelist))
		if entry, exists := cm.cache[cacheKey]; exists && !entry.Timestamp.IsZero() && time.Since(entry.Timestamp) < cm.duration {
			allReleases = append(allReleases, entry.Releases...)
			cacheHit = true
		} else {
			missingMonths = append(missingMonths, month)
		}
	}
	cm.mu.Unlock()

	// Записываем метрики кэша
	if cacheHit {
		if cm.metrics != nil {
			cm.metrics.RecordCacheHit()
		}
	} else {
		if cm.metrics != nil {
			cm.metrics.RecordCacheMiss()
		}
	}

	var filteredReleases []release.Release
	for _, rel := range allReleases {
		artistLower := strings.ToLower(rel.Artist)
		if _, ok := whitelist[artistLower]; ok {
			filteredReleases = append(filteredReleases, rel)
		}
	}

	return filteredReleases, missingMonths, nil
}

// StoreReleases stores releases for a month in the cache
func (cm *Manager) StoreReleases(month string, releases []release.Release) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var fullWhitelist []string
	fullWhitelist = append(fullWhitelist, cm.artistList.GetUnitedWhitelist()...)
	sort.Strings(fullWhitelist)
	cacheKey := fmt.Sprintf("%s-%s", strings.ToLower(month), cm.HashWhitelist(fullWhitelist))

	entry := Entry{
		Releases:  releases,
		Timestamp: time.Now(),
	}

	cm.SetEntry(cacheKey, entry)
}

// ScheduleUpdate schedules a cache update for specified months using worker pool
func (cm *Manager) ScheduleUpdate(ctx context.Context) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Проверяем, не слишком ли часто запрашивается обновление
	if time.Since(cm.lastUpdateTime) < 30*time.Second {
		cm.logger.Debug("Cache update requested too frequently, using debounce",
			zap.Duration("time_since_last", time.Since(cm.lastUpdateTime)))

		// Отменяем предыдущий таймер если он существует
		if cm.updateDebounce != nil {
			cm.updateDebounce.Stop()
		}

		// Создаем новый таймер для дебаунса
		cm.updateDebounce = time.AfterFunc(30*time.Second, func() {
			cm.mu.Lock()
			cm.isUpdating = false // Сбрасываем флаг для следующего обновления
			cm.mu.Unlock()
			cm.ScheduleUpdate(ctx)
		})
		return
	}

	if cm.isUpdating {
		cm.logger.Debug("Cache update already in progress, skipping")
		return
	}

	cm.isUpdating = true
	cm.lastUpdateTime = time.Now()
	cm.logger.Info("Scheduled cache update")

	// Создаем задачу для обновления кэша
	job := worker.Job{
		UpdateID: 0, // Не используется для cache
		UserID:   0, // Не используется для cache
		Command:  "cache_update",
		Handler: func() error {
			// Используем переданный контекст с таймаутом для предотвращения зависания
			updateCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
			defer cancel()

			if err := cm.updater.InitializeCache(updateCtx); err != nil {
				cm.logger.Error("Cache update failed", zap.Error(err))
				return err
			}

			// Сбрасываем флаг обновления после завершения
			cm.mu.Lock()
			cm.isUpdating = false
			cm.mu.Unlock()

			return nil
		},
	}

	if err := cm.workerPool.Submit(job); err != nil {
		cm.logger.Error("Failed to submit cache update job", zap.Error(err))
		// Сбрасываем флаг в случае ошибки
		cm.mu.Lock()
		cm.isUpdating = false
		cm.mu.Unlock()
	}
}

// Clear clears the cache
func (cm *Manager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache = make(map[string]Entry)
}

// StartUpdater starts the periodic cache updater using worker pool
func (cm *Manager) StartUpdater(ctx context.Context) {
	ticker := time.NewTicker(cm.duration)
	defer ticker.Stop()

	// Немедленное обновление кэша при старте
	cm.ScheduleUpdate(ctx)

	for {
		select {
		case <-ctx.Done():
			cm.logger.Info("Cache updater stopped due to context cancellation")
			return
		case <-ticker.C:
			cm.ScheduleUpdate(ctx)
		}
	}
}

// GetCachedLinks retrieves cached links for a month
func (cm *Manager) GetCachedLinks(month string) ([]string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var fullWhitelist []string
	fullWhitelist = append(fullWhitelist, cm.artistList.GetUnitedWhitelist()...)
	sort.Strings(fullWhitelist)
	cacheKey := fmt.Sprintf("%s-%s", strings.ToLower(month), cm.HashWhitelist(fullWhitelist))

	if entry, exists := cm.cache[cacheKey]; exists && !entry.Timestamp.IsZero() && time.Since(entry.Timestamp) < cm.duration {
		return entry.Links, nil
	}
	return nil, nil
}

// IsUpdating checks if an update is in progress for a month
func (cm *Manager) IsUpdating(month string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	_, exists := cm.pendingUpdates[strings.ToLower(month)]
	return exists
}

// SetEntry sets a cache entry
func (cm *Manager) SetEntry(key string, entry Entry) {
	cm.cache[key] = entry
}

// CleanupOldCacheEntries removes old cache entries
func (cm *Manager) CleanupOldCacheEntries() {
	for key, entry := range cm.cache {
		if time.Since(entry.Timestamp) > cm.duration {
			delete(cm.cache, key)
		}
	}
}

// HashWhitelist generates a hash of the whitelist
func (cm *Manager) HashWhitelist(whitelist []string) string {
	sort.Strings(whitelist)
	hash := sha256.Sum256([]byte(strings.Join(whitelist, ",")))
	return hex.EncodeToString(hash[:])
}

// GetCachedReleasesCount возвращает общее количество релизов в кэше
func (cm *Manager) GetCachedReleasesCount() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	totalReleases := 0
	for _, entry := range cm.cache {
		totalReleases += len(entry.Releases)
	}
	return totalReleases
}
