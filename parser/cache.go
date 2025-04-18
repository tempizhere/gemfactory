package parser

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"new_parser/models"
	"new_parser/utils"

	"go.uber.org/zap"
)

// CacheEntry holds cached releases
type CacheEntry struct {
	Releases  []models.Release
	Timestamp time.Time
}

var cache = make(map[string]CacheEntry)
var cacheMu sync.RWMutex

// CacheKeyInfo holds information about a cached key
type CacheKeyInfo struct {
	Months        []string
	Whitelist     map[string]struct{}
	FullWhitelist map[string]struct{}
}

// cacheKeys holds the list of cache keys and their associated parameters
var cacheKeys = make(map[string]CacheKeyInfo)
var cacheKeysMu sync.RWMutex

// InitializeCache initializes the cache for all months asynchronously
func InitializeCache(logger *zap.Logger) {
	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	maxRetries := utils.GetMaxRetries()
	delay := utils.GetRequestDelay()

	for _, month := range months {
		go func(month string) {
			for retries := 0; retries < maxRetries; retries++ {
				logger.Info("Initializing cache for month", zap.String("month", month), zap.Int("retry", retries))
				fullWhitelist := utils.LoadWhitelist(false)
				_, err := GetReleasesForMonths([]string{month}, fullWhitelist, false, false, fullWhitelist, logger)
				if err != nil {
					logger.Error("Failed to initialize cache for month", zap.String("month", month), zap.Error(err))
					if retries < maxRetries-1 {
						logger.Info("Retrying cache initialization after delay", zap.String("month", month), zap.Duration("delay", delay))
						time.Sleep(delay)
						continue
					}
					logger.Error("Max retries reached for cache initialization", zap.String("month", month))
				}
				break
			}
		}(month)
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

	cacheKeysMu.Lock()
	defer cacheKeysMu.Unlock()
	cacheKeys = make(map[string]CacheKeyInfo)
}

// UpdateCache updates expired cache entries, one at a time
func UpdateCache(logger *zap.Logger) {
	cacheKeysMu.RLock()
	keys := make([]string, 0, len(cacheKeys))
	for key := range cacheKeys {
		keys = append(keys, key)
	}
	cacheKeysMu.RUnlock()

	// Проверяем, есть ли устаревшие ключи
	cacheDurationStr := os.Getenv("CACHE_DURATION")
	cacheDuration, err := time.ParseDuration(cacheDurationStr)
	if err != nil || cacheDuration <= 0 {
		cacheDuration = 24 * time.Hour
	}

	// Ищем первый устаревший ключ и обновляем только его
	for _, key := range keys {
		cacheMu.RLock()
		entry, exists := cache[key]
		cacheMu.RUnlock()

		if !exists {
			continue
		}

		if time.Since(entry.Timestamp) >= cacheDuration {
			cacheKeysMu.RLock()
			info, ok := cacheKeys[key]
			cacheKeysMu.RUnlock()

			if !ok {
				continue
			}

			logger.Info("Cache expired, updating", zap.String("months", strings.Join(info.Months, ",")))
			releases, err := GetReleasesForMonths(info.Months, info.Whitelist, false, false, info.FullWhitelist, logger)
			if err != nil {
				logger.Error("Failed to update cache", zap.String("months", strings.Join(info.Months, ",")), zap.Error(err))
				return // Прерываем цикл после первой неудачной попытки, попробуем снова в следующем цикле
			}

			cacheMu.Lock()
			logger.Info("Updating cache", zap.String("months", strings.Join(info.Months, ",")))
			cache[key] = CacheEntry{Releases: releases, Timestamp: time.Now()}
			cacheMu.Unlock()
			logger.Info("Cache updated", zap.String("months", strings.Join(info.Months, ",")))
			return // Обновили один ключ, выходим из цикла
		}
	}
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

	// Чтение CACHE_DURATION из .env
	cacheDurationStr := os.Getenv("CACHE_DURATION")
	cacheDuration, err := time.ParseDuration(cacheDurationStr)
	if err != nil || cacheDuration <= 0 {
		cacheDuration = 24 * time.Hour // Значение по умолчанию
	}

	// Проверяем кэш для текущего whitelist
	cacheMu.RLock()
	if entry, ok := cache[cacheKey]; ok && time.Since(entry.Timestamp) < cacheDuration {
		cacheMu.RUnlock()
		logger.Info("Returning releases from cache", zap.String("months", strings.Join(months, ",")), zap.Int("whitelist_size", len(whitelist)))
		return entry.Releases, nil
	}
	cacheMu.RUnlock()

	// Если femaleOnly=true или maleOnly=true, проверяем кэш для полного whitelist
	if (femaleOnly || maleOnly) && fullWhitelist != nil {
		fullCacheKey := fmt.Sprintf("%s-%v", strings.Join(months, ","), fullWhitelist)
		cacheMu.RLock()
		if entry, ok := cache[fullCacheKey]; ok && time.Since(entry.Timestamp) < cacheDuration {
			cacheMu.RUnlock()
			logger.Info("Returning filtered releases from full whitelist cache", zap.String("months", strings.Join(months, ",")), zap.Int("whitelist_size", len(whitelist)), zap.Int("full_whitelist_size", len(fullWhitelist)))
			return FilterReleasesByWhitelist(entry.Releases, whitelist), nil
		}
		cacheMu.RUnlock()
	}

	// Логируем обновление кэша из-за протухания или отсутствия
	logger.Info("Cache expired or not found, updating cache", zap.String("months", strings.Join(months, ",")), zap.Int("whitelist_size", len(whitelist)))

	// Если кэш не найден, парсим сайт
	monthlyLinks, err := GetMonthlyLinks(months, logger)
	if err != nil {
		logger.Error("Failed to update cache", zap.Error(err))
		// Проверяем, есть ли устаревшие данные в кэше
		cacheMu.RLock()
		if entry, ok := cache[cacheKey]; ok {
			cacheMu.RUnlock()
			logger.Warn("Returning stale cache data due to fetch error", zap.String("months", strings.Join(months, ",")), zap.Int("whitelist_size", len(whitelist)))
			return entry.Releases, nil
		}
		cacheMu.RUnlock()
		return nil, fmt.Errorf("failed to get monthly links: %v", err)
	}

	var allReleases []models.Release
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, link := range monthlyLinks {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			releases, err := ParseMonthlyPage(link, whitelist, months[0], logger)
			if err != nil {
				logger.Error("Failed to parse page", zap.String("url", link), zap.Error(err))
				return
			}
			mu.Lock()
			allReleases = append(allReleases, releases...)
			mu.Unlock()
		}(link)
	}

	wg.Wait()

	sort.Slice(allReleases, func(i, j int) bool {
		dateI, _ := time.Parse(models.DateFormat, allReleases[i].Date)
		dateJ, _ := time.Parse(models.DateFormat, allReleases[j].Date)
		return dateI.Before(dateJ)
	})

	cacheMu.Lock()
	cache[cacheKey] = CacheEntry{Releases: allReleases, Timestamp: time.Now()}
	cacheMu.Unlock()

	// Сохраняем параметры запроса для фонового обновления
	cacheKeysMu.Lock()
	cacheKeys[cacheKey] = CacheKeyInfo{
		Months:        months,
		Whitelist:     whitelist,
		FullWhitelist: fullWhitelist,
	}
	cacheKeysMu.Unlock()

	return allReleases, nil
}
