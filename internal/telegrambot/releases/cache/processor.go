package cache

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/scraper"
	"gemfactory/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

	months := release.Months
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
				logger.Info("No releases found for month, skipping cache update", zap.String("month", month), zap.Duration("duration", duration))

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
