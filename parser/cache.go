package parser

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gemfactory/models"
	"gemfactory/utils"

	"go.uber.org/zap"
)

// CacheEntry holds cached releases
type CacheEntry struct {
	Releases  []models.Release
	Timestamp time.Time
}

var cache = make(map[string]CacheEntry)
var cacheMu sync.RWMutex

// lastFullUpdate tracks the time of the last full cache update
var lastFullUpdate time.Time
var lastFullUpdateMu sync.RWMutex

// cacheDuration holds the parsed CACHE_DURATION value
var cacheDuration time.Duration

var isUpdatingCache bool
var isUpdatingCacheMu sync.Mutex

// init initializes the cache duration
func init() {
	cacheDurationStr := os.Getenv("CACHE_DURATION")
	var err error
	cacheDuration, err = time.ParseDuration(cacheDurationStr)
	if err != nil || cacheDuration <= 0 {
		cacheDuration = 24 * time.Hour // Значение по умолчанию
	}

	// Инициализируем lastFullUpdate текущим временем
	lastFullUpdateMu.Lock()
	lastFullUpdate = time.Now()
	lastFullUpdateMu.Unlock()
}

// InitializeCache initializes the cache for all months asynchronously
func InitializeCache(logger *zap.Logger) {
	isUpdatingCacheMu.Lock()
	if isUpdatingCache {
		logger.Warn("Cache update already in progress, skipping...")
		isUpdatingCacheMu.Unlock()
		return
	}
	isUpdatingCache = true
	isUpdatingCacheMu.Unlock()

	defer func() {
		isUpdatingCacheMu.Lock()
		isUpdatingCache = false
		isUpdatingCacheMu.Unlock()
	}()

	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	maxRetries := utils.GetMaxRetries()
	delay := utils.GetRequestDelay()

	logger.Info("Starting cache initialization for all months", zap.Int("month_count", len(months)))
	var wg sync.WaitGroup
	totalReleases := 0
	var totalReleasesMu sync.Mutex

	for _, month := range months {
		wg.Add(1)
		go func(month string) {
			defer wg.Done()
			var err error
			for retries := 0; retries < maxRetries; retries++ {
				fullWhitelist := utils.LoadWhitelist(false)
				releases, err := GetReleasesForMonths([]string{month}, fullWhitelist, false, false, fullWhitelist, logger)
				if err != nil {
					if retries < maxRetries-1 {
						time.Sleep(delay)
						continue
					}
					logger.Error("Max retries reached for cache initialization", zap.String("month", month), zap.Error(err))
				} else {
					totalReleasesMu.Lock()
					totalReleases += len(releases)
					totalReleasesMu.Unlock()
				}
				break
			}
			if err != nil {
				logger.Warn("Cache initialization for month completed with error", zap.String("month", month), zap.Error(err))
			}
		}(month)
	}

	wg.Wait()

	// Обновляем время последнего полного обновления
	lastFullUpdateMu.Lock()
	lastFullUpdate = time.Now()
	lastFullUpdateMu.Unlock()

	// Логируем результат
	if totalReleases == 0 {
		logger.Warn("Cache initialization completed, but no releases were added")
	} else {
		logger.Info("Cache initialization completed successfully", zap.Int("total_releases", totalReleases))
	}
}

// FilterReleasesByWhitelist filters releases by the provided whitelist
func FilterReleasesByWhitelist(releases []models.Release, whitelist map[string]struct{}) []models.Release {
	var filtered []models.Release
	for _, release := range releases {
		artistKey := strings.ToLower(release.Artist)
		if _, ok := whitelist[artistKey]; ok {
			filtered = append(filtered, release)
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
func GetReleasesForMonths(months []string, whitelist map[string]struct{}, femaleOnly, maleOnly bool, fullWhitelist map[string]struct{}, logger *zap.Logger) ([]models.Release, error) {
	if len(whitelist) == 0 {
		logger.Error("Whitelist is empty")
		return nil, fmt.Errorf("whitelist is empty")
	}

	// Если months пустой, используем текущий месяц
	if len(months) == 0 {
		months = []string{strings.ToLower(time.Now().Format("January"))}
	}

	// Ключ кэша для текущего whitelist
	cacheKey := fmt.Sprintf("%s-%v", strings.Join(months, ","), whitelist)

	cacheMu.RLock()
	entry, ok := cache[cacheKey]
	cacheMu.RUnlock()

	if ok && time.Since(entry.Timestamp) < cacheDuration {
		return FilterReleasesByWhitelist(entry.Releases, whitelist), nil
	}

	if !ok && (femaleOnly || maleOnly) && fullWhitelist != nil {
		fullCacheKey := fmt.Sprintf("%s-%v", strings.Join(months, ","), fullWhitelist)
		cacheMu.RLock()
		entry, ok = cache[fullCacheKey]
		cacheMu.RUnlock()
	}
	isFresh := ok && time.Since(entry.Timestamp) < cacheDuration

	if isFresh {
		return FilterReleasesByWhitelist(entry.Releases, whitelist), nil
	}

	// Если кэш не найден, парсим сайт
	monthlyLinks, err := GetMonthlyLinks(months, logger)
	if err != nil {
		// Проверяем, есть ли устаревшие данные в кэше
		cacheMu.RLock()
		if entry, ok := cache[cacheKey]; ok {
			cacheMu.RUnlock()
			logger.Warn("Returning stale cache data due to fetch error", zap.String("months", strings.Join(months, ",")), zap.Int("whitelist_size", len(whitelist)))
			return entry.Releases, nil
		}
		cacheMu.RUnlock()
		logger.Error("Failed to get monthly links", zap.Error(err))
		return nil, fmt.Errorf("failed to get monthly links: %v", err)
	}

	// Используем канал для сбора релизов из горутин
	releaseChan := make(chan []models.Release, len(monthlyLinks))
	var wg sync.WaitGroup

	for _, link := range monthlyLinks {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			releases, err := ParseMonthlyPage(link, whitelist, months[0], logger)
			if err != nil {
				logger.Error("Failed to parse page", zap.String("url", link), zap.Error(err))
				releaseChan <- nil
				return
			}
			releaseChan <- releases
		}(link)
	}

	// Закрываем канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(releaseChan)
	}()

	// Собираем релизы из канала
	var allReleases []models.Release
	for releases := range releaseChan {
		if releases != nil {
			allReleases = append(allReleases, releases...)
		}
	}

	sort.Slice(allReleases, func(i, j int) bool {
		dateI, _ := time.Parse(models.DateFormat, allReleases[i].Date)
		dateJ, _ := time.Parse(models.DateFormat, allReleases[j].Date)
		return dateI.Before(dateJ)
	})

	cacheMu.Lock()
	cache[cacheKey] = CacheEntry{Releases: allReleases, Timestamp: time.Now()}
	cacheMu.Unlock()

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

// Добавьте вспомогательную функцию для получения CACHE_DURATION
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
