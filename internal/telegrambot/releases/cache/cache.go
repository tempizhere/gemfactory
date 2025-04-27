package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// cacheUpdateMu protects the cache update timer
var cacheUpdateMu sync.Mutex

// cacheUpdateTimer holds the timer for delayed cache updates
var cacheUpdateTimer *time.Timer

// ClearCache clears the entire cache
func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = make(map[string]CacheEntry)
}

// ScheduleCacheUpdate schedules a cache update with a 60-second delay, resetting the timer if called again
func ScheduleCacheUpdate(config *config.Config, logger *zap.Logger, al *artistlist.ArtistList) {
	cacheUpdateMu.Lock()
	defer cacheUpdateMu.Unlock()

	// Отменяем существующий таймер, если он есть
	if cacheUpdateTimer != nil {
		cacheUpdateTimer.Stop()
	}

	// Создаём новый таймер на 60 секунд
	cacheUpdateTimer = time.AfterFunc(60*time.Second, func() {
		logger.Info("Starting delayed cache update")
		InitializeCache(config, logger, al)
		cacheUpdateMu.Lock()
		cacheUpdateTimer = nil
		cacheUpdateMu.Unlock()
	})

	logger.Info("Scheduled cache update in 60 seconds")
}

// GetReleasesForMonths retrieves releases for multiple months
func GetReleasesForMonths(months []string, whitelist map[string]struct{}, femaleOnly, maleOnly bool, al *artistlist.ArtistList, config *config.Config, logger *zap.Logger) ([]release.Release, error) {
	if logger.Core().Enabled(zapcore.DebugLevel) {
		logger.Debug("Entering GetReleasesForMonths", zap.Strings("months", months), zap.Bool("femaleOnly", femaleOnly), zap.Bool("maleOnly", maleOnly))
	}
	if len(whitelist) == 0 {
		logger.Error("Whitelist is empty")
		return nil, fmt.Errorf("whitelist is empty")
	}

	// Если months пустой, используем текущий месяц
	if len(months) == 0 {
		months = []string{strings.ToLower(time.Now().Format("January"))}
		if logger.Core().Enabled(zapcore.DebugLevel) {
			logger.Debug("No months specified, using current month", zap.Strings("months", months))
		}
	}

	// Собираем релизы для каждого месяца отдельно
	var allReleases []release.Release
	fullWhitelist := al.GetUnitedWhitelist()
	if logger.Core().Enabled(zapcore.DebugLevel) {
		logger.Debug("Comparing whitelists", zap.Int("input_whitelist_size", len(whitelist)), zap.Int("full_whitelist_size", len(fullWhitelist)))
	}

	for _, month := range months {
		cacheKey := fmt.Sprintf("%s-%s", month, hashWhitelist(fullWhitelist))
		if logger.Core().Enabled(zapcore.DebugLevel) {
			logger.Debug("Checking cache", zap.String("cache_key", cacheKey), zap.String("month", month))
		}

		cacheMu.RLock()
		entry, ok := cache[cacheKey]
		cacheMu.RUnlock()

		// Используем кэш, если данные существуют
		if ok {
			if logger.Core().Enabled(zapcore.DebugLevel) {
				logger.Debug("Using cached releases", zap.String("cache_key", cacheKey), zap.Int("release_count", len(entry.Releases)), zap.Time("cache_timestamp", entry.Timestamp))
			}
			allReleases = append(allReleases, FilterReleasesByWhitelist(entry.Releases, whitelist)...)
			continue
		}

		if logger.Core().Enabled(zapcore.DebugLevel) {
			logger.Debug("Cache entry missing", zap.String("cache_key", cacheKey))
		}

		// Если кэш отсутствует, пробуем обновить кэш до двух раз
		for attempt := 0; attempt < 2; attempt++ {
			logger.Info("Cache missing, initializing full cache update", zap.String("cache_key", cacheKey), zap.Int("attempt", attempt+1))
			InitializeCache(config, logger, al)
			cacheMu.RLock()
			entry, ok = cache[cacheKey]
			cacheMu.RUnlock()

			if ok {
				if logger.Core().Enabled(zapcore.DebugLevel) {
					logger.Debug("Using freshly updated cache", zap.String("cache_key", cacheKey), zap.Int("release_count", len(entry.Releases)), zap.Time("cache_timestamp", entry.Timestamp))
				}
				allReleases = append(allReleases, FilterReleasesByWhitelist(entry.Releases, whitelist)...)
				break
			}
			logger.Warn("No releases available after cache update for month", zap.String("month", month), zap.Int("attempt", attempt+1), zap.String("cache_key", cacheKey))
		}

		if !ok {
			logger.Warn("No releases available after retries for month", zap.String("month", month), zap.String("cache_key", cacheKey))
		}

		// Логируем содержимое кэша после попытки
		cacheMu.RLock()
		if len(cache) == 0 {
			logger.Warn("Cache is empty after GetReleasesForMonths attempt")
		} else {
			if logger.Core().Enabled(zapcore.DebugLevel) {
				logger.Debug("Cache contents after attempt", zap.Int("cache_size", len(cache)))
				for key, entry := range cache {
					logger.Debug("Cache entry", zap.String("key", key), zap.Int("release_count", len(entry.Releases)), zap.Time("timestamp", entry.Timestamp))
				}
			}
		}
		cacheMu.RUnlock()
	}

	if len(allReleases) == 0 {
		logger.Warn("No releases found for requested months", zap.Strings("months", months))
		return []release.Release{}, nil
	}

	if logger.Core().Enabled(zapcore.DebugLevel) {
		logger.Debug("Returning releases", zap.Int("release_count", len(allReleases)))
	}
	return allReleases, nil
}

// CleanupOldCacheEntries removes old cache entries
func CleanupOldCacheEntries() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	for key, entry := range cache {
		if time.Since(entry.Timestamp) > cacheDuration {
			delete(cache, key)
		}
	}
}

// GetCacheDuration returns the cache duration
func GetCacheDuration() time.Duration {
	return cacheDuration
}

// GetCacheKeys returns all cache keys
func GetCacheKeys() []string {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	keys := make([]string, 0, len(cache))
	for key := range cache {
		keys = append(keys, key)
	}
	return keys
}

// StartUpdater periodically updates the cache
func StartUpdater(config *config.Config, logger *zap.Logger, al *artistlist.ArtistList) {
	logger.Info("Starting cache updater", zap.Duration("cache_duration", cacheDuration))
	ticker := time.NewTicker(cacheDuration)
	defer ticker.Stop()

	// Немедленное синхронное обновление кэша при старте
	logger.Info("Starting initial cache update")
	InitializeCache(config, logger, al)

	for t := range ticker.C {
		logger.Info("Starting periodic cache update", zap.Time("tick_time", t))
		go InitializeCache(config, logger, al)
		logger.Info("Periodic cache update completed")
	}
}

// hashWhitelist creates a compact hash of the whitelist for cache key
func hashWhitelist(whitelist map[string]struct{}) string {
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
