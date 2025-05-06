package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gemfactory/internal/telegrambot/releases/release"
)

// GetReleasesForMonths retrieves releases for multiple months from the cache
func (cm *CacheManager) GetReleasesForMonths(months []string, whitelist map[string]struct{}, femaleOnly, maleOnly bool) ([]release.Release, []string, error) {
	if cm.logger.Core().Enabled(zapcore.DebugLevel) {
		cm.logger.Debug("Entering GetReleasesForMonths", zap.Strings("months", months), zap.Bool("femaleOnly", femaleOnly), zap.Bool("maleOnly", maleOnly))
	}
	if len(whitelist) == 0 {
		cm.logger.Error("Whitelist is empty")
		return nil, nil, fmt.Errorf("whitelist is empty")
	}

	// Если months пустой, используем текущий месяц
	if len(months) == 0 {
		months = []string{strings.ToLower(time.Now().Format("January"))}
		if cm.logger.Core().Enabled(zapcore.DebugLevel) {
			cm.logger.Debug("No months specified, using current month", zap.Strings("months", months))
		}
	}

	// Собираем релизы из кэша
	var allReleases []release.Release
	missingMonths := make([]string, 0)
	fullWhitelist := cm.artistList.GetUnitedWhitelist()
	if cm.logger.Core().Enabled(zapcore.DebugLevel) {
		cm.logger.Debug("Comparing whitelists", zap.Int("input_whitelist_size", len(whitelist)), zap.Int("full_whitelist_size", len(fullWhitelist)))
	}

	cm.mu.RLock()
	for _, month := range months {
		cacheKey := fmt.Sprintf("%s-%s", month, HashWhitelist(fullWhitelist))
		if cm.logger.Core().Enabled(zapcore.DebugLevel) {
			cm.logger.Debug("Checking cache", zap.String("cache_key", cacheKey), zap.String("month", month))
		}

		if entry, ok := cm.cache[cacheKey]; ok {
			if cm.logger.Core().Enabled(zapcore.DebugLevel) {
				cm.logger.Debug("Using cached releases", zap.String("cache_key", cacheKey), zap.Int("release_count", len(entry.Releases)), zap.Time("cache_timestamp", entry.Timestamp))
			}
			allReleases = append(allReleases, FilterReleasesByWhitelist(entry.Releases, whitelist)...)
		} else {
			missingMonths = append(missingMonths, month)
		}
	}
	cm.mu.RUnlock()

	// Если есть отсутствующие месяцы, планируем асинхронное обновление
	if len(missingMonths) > 0 {
		if cm.logger.Core().Enabled(zapcore.DebugLevel) {
			cm.logger.Debug("No cache for months, scheduling update", zap.Strings("months", missingMonths))
		}
		go cm.ScheduleUpdateWithMonths(missingMonths)
	}

	if cm.logger.Core().Enabled(zapcore.DebugLevel) {
		cm.logger.Debug("Returning releases", zap.Int("release_count", len(allReleases)))
	}
	return allReleases, missingMonths, nil
}

// ScheduleUpdate schedules a cache update with a 60-second delay
func (cm *CacheManager) ScheduleUpdate() {
	cm.ScheduleUpdateWithMonths(nil)
}

// ScheduleUpdateWithMonths schedules a cache update for specific months with a 60-second delay
func (cm *CacheManager) ScheduleUpdateWithMonths(months []string) {
	cm.updateTimerMu.Lock()
	defer cm.updateTimerMu.Unlock()

	if cm.isUpdating {
		cm.logger.Debug("Update already scheduled or in progress, skipping")
		return
	}

	if cm.updateTimer != nil {
		cm.updateTimer.Stop()
	}

	cm.isUpdating = true
	// Отмечаем месяцы как обновляемые и сохраняем время
	now := time.Now()
	for _, month := range months {
		cm.pendingUpdates[month] = struct{}{}
		cm.pendingUpdatesTimestamps[month] = now
	}

	cm.updateTimer = time.AfterFunc(60*time.Second, func() {
		cm.logger.Info("Starting delayed cache update", zap.Strings("months", months))
		const maxRetries = 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err := cm.updater.InitializeCache(context.Background()); err != nil {
				cm.logger.Error("Failed to initialize cache", zap.Error(err), zap.Int("attempt", attempt))
				if attempt < maxRetries {
					time.Sleep(time.Duration(attempt) * time.Second)
					continue
				}
				cm.logger.Warn("Cache update failed after retries")
			}
			break
		}
		cm.updateTimerMu.Lock()
		cm.updateTimer = nil
		cm.isUpdating = false
		// Очищаем pendingUpdates и pendingUpdatesTimestamps после обновления
		for _, month := range months {
			delete(cm.pendingUpdates, month)
			delete(cm.pendingUpdatesTimestamps, month)
		}
		cm.updateTimerMu.Unlock()
	})

	cm.logger.Info("Scheduled cache update", zap.Strings("months", months))
}

// IsUpdating checks if the cache is being updated for a specific month
func (cm *CacheManager) IsUpdating(month string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Проверяем, есть ли месяц в pendingUpdates
	_, updating := cm.pendingUpdates[month]
	if !updating {
		return false
	}

	// Проверяем таймаут (5 минут)
	timestamp, exists := cm.pendingUpdatesTimestamps[month]
	if exists && time.Since(timestamp) > 5*time.Minute {
		// Удаляем устаревшую запись
		delete(cm.pendingUpdates, month)
		delete(cm.pendingUpdatesTimestamps, month)
		cm.logger.Debug("Removed stale pending update", zap.String("month", month))
		return false
	}

	return true
}

// Clear clears the entire cache
func (cm *CacheManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cache = make(map[string]CacheEntry)
	// Очищаем pendingUpdates и pendingUpdatesTimestamps при очистке кэша
	for month := range cm.pendingUpdates {
		delete(cm.pendingUpdates, month)
		delete(cm.pendingUpdatesTimestamps, month)
	}
}

// CleanupOldCacheEntries removes old cache entries
func (cm *CacheManager) CleanupOldCacheEntries() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.isUpdating {
		cm.logger.Debug("Cache update in progress, skipping cleanup")
		return
	}

	for key, entry := range cm.cache {
		if time.Since(entry.Timestamp) > cm.getCacheDuration(key) {
			delete(cm.cache, key)
		}
	}
}

// getCacheDuration returns the cache duration based on the month
func (cm *CacheManager) getCacheDuration(cacheKey string) time.Duration {
	for _, month := range GetActiveMonths() {
		if strings.HasPrefix(cacheKey, month+"-") {
			return cm.duration // 8 hours for active months
		}
	}
	return 24 * time.Hour // 24 hours for inactive months
}

// StartUpdater starts periodic cache updates
func (cm *CacheManager) StartUpdater() {
	cm.updater.StartUpdater()
}

// GetCachedLinks retrieves cached links for a month
func (cm *CacheManager) GetCachedLinks(month string) ([]string, error) {
	cacheKey := fmt.Sprintf("links-%s", month)
	cm.mu.RLock()
	entry, ok := cm.cache[cacheKey]
	cm.mu.RUnlock()
	if ok && time.Since(entry.Timestamp) < cm.duration {
		cm.logger.Debug("Using cached links", zap.String("month", month), zap.Int("link_count", len(entry.Links)))
		return entry.Links, nil
	}

	links, err := cm.scraper.GetMonthlyLinksWithContext(context.Background(), []string{month}, cm.config, cm.logger)
	if err != nil {
		cm.logger.Error("Failed to fetch links", zap.String("month", month), zap.Error(err))
		return nil, err
	}

	cm.mu.Lock()
	cm.cache[cacheKey] = CacheEntry{Links: links, Timestamp: time.Now()}
	cm.mu.Unlock()
	cm.logger.Info("Cached links", zap.String("month", month), zap.Int("link_count", len(links)))
	return links, nil
}

// StoreReleases stores releases in the cache
func (cm *CacheManager) StoreReleases(month string, releases []release.Release) {
	cacheKey := fmt.Sprintf("%s-%s", month, HashWhitelist(cm.artistList.GetUnitedWhitelist()))
	cm.mu.Lock()
	cm.cache[cacheKey] = CacheEntry{
		Releases:  releases,
		Timestamp: time.Now(),
	}
	cm.mu.Unlock()
}

// HashWhitelist creates a compact hash of the whitelist for cache key
func HashWhitelist(whitelist map[string]struct{}) string {
	var keys []string
	for key := range whitelist {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hasher := sha256.New()
	hasher.Write([]byte(strings.Join(keys, ",")))
	return hex.EncodeToString(hasher.Sum(nil))[:8]
}

// FilterReleasesByWhitelist filters releases by the provided whitelist
func FilterReleasesByWhitelist(releases []release.Release, whitelist map[string]struct{}) []release.Release {
	var filtered []release.Release
	for _, rel := range releases {
		artistKey := strings.ToLower(rel.Artist)
		if _, ok := whitelist[artistKey]; ok {
			filtered = append(filtered, rel)
		}
	}
	return filtered
}

// GetActiveMonths returns the list of active months (previous, current, next)
func GetActiveMonths() []string {
	now := time.Now()
	return []string{
		strings.ToLower(now.AddDate(0, -1, 0).Format("January")),
		strings.ToLower(now.Format("January")),
		strings.ToLower(now.AddDate(0, 1, 0).Format("January")),
	}
}
