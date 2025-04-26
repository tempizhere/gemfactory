package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gemfactory/internal/features/releasesbot/artistlist"
	"gemfactory/internal/features/releasesbot/release"
	"gemfactory/internal/features/releasesbot/scraper"
	"gemfactory/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CacheEntry holds cached releases
type CacheEntry struct {
	Releases  []release.Release
	Timestamp time.Time
}

var cache = make(map[string]CacheEntry)
var cacheMu sync.RWMutex

// cacheDuration holds the parsed CACHE_DURATION value
var cacheDuration time.Duration
var cacheDurationOnce sync.Once

var activeUpdates int
var activeUpdatesMu sync.Mutex

// InitCacheConfig initializes the cache configuration
func InitCacheConfig(logger *zap.Logger) {
	cacheDurationOnce.Do(func() {
		cacheDurationStr := os.Getenv("CACHE_DURATION")
		if cacheDurationStr == "" {
			logger.Warn("CACHE_DURATION not set, using default", zap.Duration("default", 24*time.Hour))
			cacheDuration = 24 * time.Hour
			return
		}

		var err error
		cacheDuration, err = time.ParseDuration(strings.TrimSpace(cacheDurationStr))
		if err != nil || cacheDuration <= 0 {
			logger.Warn("Invalid CACHE_DURATION, using default", zap.String("value", cacheDurationStr), zap.Error(err), zap.Duration("default", 24*time.Hour))
			cacheDuration = 24 * time.Hour
			return
		}
		logger.Info("CACHE_DURATION parsed successfully", zap.String("value", cacheDurationStr), zap.Duration("duration", cacheDuration))
	})
}

// InitializeCache initializes the cache for all months asynchronously
func InitializeCache(config *config.Config, logger *zap.Logger, al *artistlist.ArtistList) {
	// Проверяем уровень логирования
	if logger.Core().Enabled(zapcore.DebugLevel) {
		logger.Debug("InitializeCache started", zap.Bool("debug_enabled", true))
	} else {
		logger.Info("InitializeCache started, debug logging disabled")
	}

	activeUpdatesMu.Lock()
	activeUpdates++
	activeUpdatesMu.Unlock()

	defer func() {
		activeUpdatesMu.Lock()
		activeUpdates--
		activeUpdatesMu.Unlock()
		if logger.Core().Enabled(zapcore.DebugLevel) {
			logger.Debug("Cache update completed, active updates", zap.Int("active_updates", activeUpdates))
		}
	}()

	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}
	monthOrder := map[string]int{
		"january":   1,
		"february":  2,
		"march":     3,
		"april":     4,
		"may":       5,
		"june":      6,
		"july":      7,
		"august":    8,
		"september": 9,
		"october":   10,
		"november":  11,
		"december":  12,
	}

	logger.Info("Starting cache initialization for all months", zap.Int("month_count", len(months)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Таймаут для всего процесса
	time.AfterFunc(8*time.Minute, func() {
		logger.Warn("Cache initialization timed out, cancelling context")
		cancel()
	})

	stop := make(chan struct{})
	defer close(stop)

	var wg sync.WaitGroup
	totalReleases := 0
	var totalReleasesMu sync.Mutex
	var successfulMonths, emptyMonths []string
	var monthsMu sync.Mutex

	// Ограничиваем количество одновременно обрабатываемых месяцев
	semaphore := make(chan struct{}, 4) // Максимум 4 месяца одновременно

	for _, month := range months {
		wg.Add(1)
		go func(month string) {
			defer wg.Done()
			// Захватываем семафор
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if logger.Core().Enabled(zapcore.DebugLevel) {
				logger.Debug("Started processing month", zap.String("month", month))
			}

			// Индивидуальный таймаут для месяца
			monthCtx, monthCancel := context.WithTimeout(ctx, 2*time.Minute)
			defer monthCancel()

			startTime := time.Now()
			var releases []release.Release
			var err error
			for retries := 0; retries < config.MaxRetries; retries++ {
				select {
				case <-monthCtx.Done():
					logger.Warn("Cache initialization cancelled for month", zap.String("month", month), zap.Error(monthCtx.Err()))
					return
				case <-stop:
					logger.Warn("Cache initialization stopped for month", zap.String("month", month))
					return
				default:
					if logger.Core().Enabled(zapcore.DebugLevel) {
						logger.Debug("Fetching monthly links", zap.String("month", month), zap.Int("retry", retries+1))
					}
					fullWhitelist := al.GetUnitedWhitelist()
					monthlyLinks, err := scraper.GetMonthlyLinksWithContext(monthCtx, []string{month}, config, logger)
					if err != nil {
						if retries < config.MaxRetries-1 {
							time.Sleep(config.RequestDelay)
							continue
						}
						logger.Error("Failed to get monthly links", zap.String("month", month), zap.Error(err))
						break
					}

					releaseChan := make(chan []release.Release, len(monthlyLinks))
					var parseWg sync.WaitGroup
					for _, link := range monthlyLinks {
						parseWg.Add(1)
						go func(link string) {
							defer func() {
								parseWg.Done()
								if logger.Core().Enabled(zapcore.DebugLevel) {
									logger.Debug("Completed parsing page", zap.String("url", link))
								}
							}()
							select {
							case <-monthCtx.Done():
								logger.Warn("Page parsing cancelled", zap.String("url", link), zap.Error(monthCtx.Err()))
								return
							case <-stop:
								logger.Warn("Page parsing stopped", zap.String("url", link))
								return
							default:
								rels, err := scraper.ParseMonthlyPageWithContext(monthCtx, link, fullWhitelist, month, config, logger)
								if err != nil {
									logger.Error("Failed to parse page", zap.String("url", link), zap.Error(err))
									releaseChan <- nil
									return
								}
								releaseChan <- rels
							}
						}(link)
					}

					go func() {
						parseWg.Wait()
						close(releaseChan)
						if logger.Core().Enabled(zapcore.DebugLevel) {
							logger.Debug("Closed release channel for month", zap.String("month", month))
						}
					}()

					var allReleases []release.Release
					for rels := range releaseChan {
						select {
						case <-monthCtx.Done():
							logger.Warn("Release collection cancelled for month", zap.String("month", month), zap.Error(monthCtx.Err()))
							return
						case <-stop:
							logger.Warn("Release collection stopped for month", zap.String("month", month))
							return
						default:
							if rels != nil {
								allReleases = append(allReleases, rels...)
							}
						}
					}

					if len(allReleases) > 0 {
						releases = allReleases
						break
					}
					if retries < config.MaxRetries-1 {
						time.Sleep(config.RequestDelay)
					}
				}
			}

			duration := time.Since(startTime)
			if err != nil {
				logger.Warn("Cache initialization for month completed with error", zap.String("month", month), zap.Error(err), zap.Duration("duration", duration))
			} else if len(releases) > 0 {
				totalReleasesMu.Lock()
				totalReleases += len(releases)
				totalReleasesMu.Unlock()

				// Сохраняем релизы в кэш
				cacheMu.Lock()
				cacheKey := fmt.Sprintf("%s-%s", month, hashWhitelist(al.GetUnitedWhitelist()))
				cache[cacheKey] = CacheEntry{
					Releases:  releases,
					Timestamp: time.Now(),
				}
				if logger.Core().Enabled(zapcore.DebugLevel) {
					logger.Debug("Cached releases for month", zap.String("month", month), zap.Int("release_count", len(releases)), zap.String("cache_key", cacheKey), zap.Duration("duration", duration))
				}
				cacheMu.Unlock()

				// Добавляем месяц в список успешных
				monthsMu.Lock()
				successfulMonths = append(successfulMonths, month)
				monthsMu.Unlock()
			} else {
				logger.Warn("No releases found for month, skipping cache update", zap.String("month", month), zap.Duration("duration", duration))

				// Добавляем месяц в список пустых
				monthsMu.Lock()
				emptyMonths = append(emptyMonths, month)
				monthsMu.Unlock()
			}
		}(month)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		logger.Info("All months processed successfully")
	}()

	select {
	case <-done:
		logger.Info("Cache initialization completed successfully")
	case <-ctx.Done():
		close(stop) // Принудительно останавливаем горутины
		logger.Warn("Cache initialization cancelled", zap.Error(ctx.Err()))
	}

	// Логируем содержимое кэша
	cacheMu.RLock()
	if len(cache) == 0 {
		logger.Warn("Cache is empty after initialization")
	} else {
		logger.Info("Cache contents", zap.Int("cache_size", len(cache)))
		for key, entry := range cache {
			logger.Info("Cache entry", zap.String("key", key), zap.Int("release_count", len(entry.Releases)), zap.Time("timestamp", entry.Timestamp))
		}
	}
	cacheMu.RUnlock()

	// Сортируем списки месяцев по хронологическому порядку
	sort.Slice(successfulMonths, func(i, j int) bool {
		return monthOrder[successfulMonths[i]] < monthOrder[successfulMonths[j]]
	})
	sort.Slice(emptyMonths, func(i, j int) bool {
		return monthOrder[emptyMonths[i]] < monthOrder[emptyMonths[j]]
	})

	// Логируем списки месяцев
	if len(successfulMonths) > 0 {
		logger.Info("Successful cache updates for months", zap.String("months", strings.Join(successfulMonths, ",")))
	} else {
		logger.Warn("No successful cache updates for any months")
	}
	if len(emptyMonths) > 0 {
		logger.Info("No releases found for months", zap.String("months", strings.Join(emptyMonths, ",")))
	} else {
		logger.Info("Releases found for all months")
	}

	// Логируем результат
	if totalReleases == 0 {
		logger.Warn("Cache initialization completed, but no releases were added")
	} else {
		logger.Info("Cache updated successfully", zap.Int("total_releases", totalReleases))
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

// ClearCache clears the entire cache
func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = make(map[string]CacheEntry)
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

	logger.Info("Cache updater stopped")
}
