package updater

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
)

// InitializeCache initializes the cache for all months asynchronously
func (u *UpdaterImpl) InitializeCache(ctx context.Context) error {
	if u.logger.Core().Enabled(zapcore.DebugLevel) {
		u.logger.Debug("InitializeCache started", zap.Bool("debug_enabled", true))
	} else {
		u.logger.Info("InitializeCache started, debug logging disabled")
	}

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

	u.logger.Info("Starting cache initialization for all months", zap.Int("month_count", len(months)))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Таймаут для всего процесса
	time.AfterFunc(15*time.Minute, func() {
		u.logger.Warn("Cache initialization timed out, cancelling context")
		cancel()
	})

	stop := make(chan struct{})
	defer close(stop)

	var wg sync.WaitGroup
	totalReleases := 0
	var totalReleasesMu sync.Mutex
	var successfulMonths, emptyMonths []string
	var monthsMu sync.Mutex

	// Обработка активных и неактивных месяцев
	activeMonths := cache.GetActiveMonths()
	inactiveMonths := make([]string, 0, len(months))
	for _, month := range months {
		if !contains(activeMonths, month) {
			inactiveMonths = append(inactiveMonths, month)
		}
	}

	// Обновление активных месяцев
	for _, month := range activeMonths {
		if !contains(months, month) {
			continue // Пропускаем месяцы, не входящие в release.Months
		}
		wg.Add(1)
		go u.processMonth(ctx, month, &wg, &totalReleases, &totalReleasesMu, &successfulMonths, &emptyMonths, &monthsMu, stop)
	}

	// Обновление неактивных месяцев
	for _, month := range inactiveMonths {
		wg.Add(1)
		go u.processMonth(ctx, month, &wg, &totalReleases, &totalReleasesMu, &successfulMonths, &emptyMonths, &monthsMu, stop)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		u.logger.Info("All months processed successfully")
	}()

	select {
	case <-done:
		u.logger.Info("Cache initialization completed successfully")
	case <-ctx.Done():
		close(stop)
		u.logger.Warn("Cache initialization cancelled", zap.Error(ctx.Err()))
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
		u.logger.Info("Successful cache updates for months", zap.String("months", strings.Join(successfulMonths, ",")))
	} else {
		u.logger.Warn("No successful cache updates for any months")
	}
	if len(emptyMonths) > 0 {
		u.logger.Debug("No releases found for months", zap.String("months", strings.Join(emptyMonths, ",")))
	} else {
		u.logger.Info("Releases found for all months")
	}

	if totalReleases == 0 {
		u.logger.Warn("Cache initialization completed, but no releases were added")
	} else {
		u.logger.Info("Cache updated successfully", zap.Int("total_releases", totalReleases))
	}

	return nil
}

// processMonth processes a single month
func (u *UpdaterImpl) processMonth(ctx context.Context, month string, wg *sync.WaitGroup, totalReleases *int, totalReleasesMu *sync.Mutex, successfulMonths, emptyMonths *[]string, monthsMu *sync.Mutex, stop chan struct{}) {
	defer wg.Done()

	if u.logger.Core().Enabled(zapcore.DebugLevel) {
		u.logger.Debug("Started processing month", zap.String("month", month))
	}

	monthCtx, monthCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer monthCancel()

	startTime := time.Now()
	var releases []release.Release
	var err error
	for retries := 0; retries < u.config.MaxRetries; retries++ {
		select {
		case <-monthCtx.Done():
			u.logger.Warn("Cache initialization cancelled for month", zap.String("month", month), zap.Error(monthCtx.Err()))
			return
		case <-stop:
			u.logger.Warn("Cache initialization stopped for month", zap.String("month", month))
			return
		default:
			if u.logger.Core().Enabled(zapcore.DebugLevel) {
				u.logger.Debug("Fetching monthly links", zap.String("month", month), zap.Int("retry", retries+1))
			}
			monthlyLinks, err := u.cache.GetCachedLinks(month)
			if err != nil {
				if retries < u.config.MaxRetries-1 {
					time.Sleep(u.config.RequestDelay)
					continue
				}
				u.logger.Error("Failed to get monthly links", zap.String("month", month), zap.Error(err))
				break
			}

			releaseChan := make(chan []release.Release, len(monthlyLinks))
			var parseWg sync.WaitGroup
			for _, link := range monthlyLinks {
				parseWg.Add(1)
				go func(link string) {
					defer func() {
						parseWg.Done()
						if u.logger.Core().Enabled(zapcore.DebugLevel) {
							u.logger.Debug("Completed parsing page", zap.String("url", link))
						}
					}()
					select {
					case <-monthCtx.Done():
						u.logger.Warn("Page parsing cancelled", zap.String("url", link), zap.Error(monthCtx.Err()))
						return
					case <-stop:
						u.logger.Warn("Page parsing stopped", zap.String("url", link))
						return
					default:
						rels, err := u.scraper.ParseMonthlyPageWithContext(monthCtx, link, u.artistList.GetUnitedWhitelist(), month, u.config, u.logger)
						if err != nil {
							u.logger.Error("Failed to parse page", zap.String("url", link), zap.Error(err))
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
				if u.logger.Core().Enabled(zapcore.DebugLevel) {
					u.logger.Debug("Closed release channel for month", zap.String("month", month))
				}
			}()

			var allReleases []release.Release
			for rels := range releaseChan {
				select {
				case <-monthCtx.Done():
					u.logger.Warn("Release collection cancelled for month", zap.String("month", month), zap.Error(monthCtx.Err()))
					return
				case <-stop:
					u.logger.Warn("Release collection stopped for month", zap.String("month", month))
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
			if retries < u.config.MaxRetries-1 {
				time.Sleep(u.config.RequestDelay)
			}
		}
	}

	duration := time.Since(startTime)
	if err != nil {
		u.logger.Warn("Cache initialization for month completed with error", zap.String("month", month), zap.Error(err), zap.Duration("duration", duration))
	} else if len(releases) > 0 {
		totalReleasesMu.Lock()
		*totalReleases += len(releases)
		totalReleasesMu.Unlock()

		// Сохраняем релизы в кэш
		u.cache.StoreReleases(month, releases)
		if u.logger.Core().Enabled(zapcore.DebugLevel) {
			u.logger.Debug("Cached releases for month", zap.String("month", month), zap.Int("release_count", len(releases)), zap.Duration("duration", duration))
		}

		// Добавляем месяц в список успешных
		monthsMu.Lock()
		*successfulMonths = append(*successfulMonths, month)
		monthsMu.Unlock()
	} else {
		u.logger.Debug("No releases found for month, skipping cache update", zap.String("month", month), zap.Duration("duration", duration))

		// Добавляем месяц в список пустых
		monthsMu.Lock()
		*emptyMonths = append(*emptyMonths, month)
		monthsMu.Unlock()
	}
}

// StartUpdater periodically updates the cache
func (u *UpdaterImpl) StartUpdater() {
	u.logger.Info("Starting cache updater", zap.Duration("cache_duration", u.config.CacheDuration))
	ticker := time.NewTicker(u.config.CacheDuration)
	defer ticker.Stop()

	// Немедленное синхронное обновление кэша при старте
	u.logger.Info("Starting initial cache update")
	if err := u.InitializeCache(context.Background()); err != nil {
		u.logger.Error("Initial cache update failed", zap.Error(err))
	}

	for t := range ticker.C {
		u.logger.Info("Starting periodic cache update", zap.Time("tick_time", t))
		go func() {
			if err := u.InitializeCache(context.Background()); err != nil {
				u.logger.Error("Periodic cache update failed", zap.Error(err))
			}
		}()
		u.logger.Info("Periodic cache update completed")
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
