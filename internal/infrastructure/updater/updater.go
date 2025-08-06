package updater

import (
	"context"
	"fmt"
	"gemfactory/internal/domain/release"
	"gemfactory/internal/infrastructure/middleware"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// formatDuration форматирует время в читаемый формат (например: 8s)
func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	return fmt.Sprintf("%ds", seconds)
}

// InitializeCache initializes the cache for all months sequentially
func (u *Impl) InitializeCache(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	// Устанавливаем статус обновления
	if u.metrics != nil {
		u.metrics.SetCacheUpdateStatus(true)
	}

	u.logger.Debug("Cache initialization configuration",
		zap.Int("max_retries", u.config.MaxRetries),
		zap.Duration("request_delay", u.config.RequestDelay))

	cfg := release.NewConfig()
	months := cfg.Months()
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

	// Fetch links for all months once
	monthLinks := make(map[string][]string)
	requestCount := 0

	// Проверяем контекст перед запросом ссылок
	select {
	case <-ctx.Done():
		u.logger.Debug("Context cancelled before fetching links", zap.Error(ctx.Err()))
		if u.metrics != nil {
			u.metrics.SetCacheUpdateStatus(false)
		}
		return ctx.Err()
	default:
	}

	links, err := u.scraper.FetchMonthlyLinks(ctx, months)
	if err != nil {
		if ctx.Err() != nil {
			u.logger.Debug("FetchMonthlyLinks cancelled due to context cancellation", zap.Error(ctx.Err()))
			if u.metrics != nil {
				u.metrics.SetCacheUpdateStatus(false)
			}
			return ctx.Err()
		}
		u.logger.Error("Failed to fetch links for months", zap.Error(err))
		if u.metrics != nil {
			u.metrics.SetCacheUpdateStatus(false)
		}
		return fmt.Errorf("failed to fetch links: %w", err)
	}
	requestCount++
	u.logger.Debug("Fetched all links", zap.Strings("links", links), zap.Int("request_count", requestCount))

	for _, link := range links {
		for _, month := range months {
			if strings.Contains(strings.ToLower(link), strings.ToLower(month)) {
				monthLinks[strings.ToLower(month)] = append(monthLinks[strings.ToLower(month)], link)
			}
		}
	}

	var successfulMonths, emptyMonths []string
	var monthsMu sync.Mutex
	totalReleases := 0
	var totalReleasesMu sync.Mutex
	var errs []error
	var errsMu sync.Mutex

	// Последовательная обработка месяцев для избежания блокировки сайта
	for _, month := range months {
		// Проверяем контекст перед обработкой каждого месяца
		select {
		case <-ctx.Done():
			u.logger.Info("Context cancelled, stopping month processing early", zap.Error(ctx.Err()))
			if u.metrics != nil {
				u.metrics.SetCacheUpdateStatus(false)
			}
			return ctx.Err()
		default:
		}

		u.logger.Debug("Starting task", zap.String("task", "process month "+month))
		monthCtx, monthCancel := context.WithCancel(ctx)
		defer monthCancel()

		err := middleware.WithTaskLogging(u.logger, "process month "+month)(
			monthCtx, u.logger,
			func() error {
				return u.processMonth(monthCtx, month, monthLinks[strings.ToLower(month)], &totalReleases, &totalReleasesMu, &successfulMonths, &emptyMonths, &monthsMu)
			},
		)
		if err != nil {
			// Если ошибка связана с отменой контекста, останавливаем обработку
			if err == context.Canceled || err == context.DeadlineExceeded {
				u.logger.Info("Month processing cancelled due to context", zap.String("month", month), zap.Error(err))
				if u.metrics != nil {
					u.metrics.SetCacheUpdateStatus(false)
				}
				return err
			}
			u.logger.Error("Failed to process month", zap.String("month", month), zap.Error(err))
			errsMu.Lock()
			errs = append(errs, fmt.Errorf("month %s: %w", month, err))
			errsMu.Unlock()
		}
	}

	// Сортируем списки месяцев по хронологическому порядку
	sort.Slice(successfulMonths, func(i, j int) bool {
		return monthOrder[successfulMonths[i]] < monthOrder[successfulMonths[j]]
	})
	sort.Slice(emptyMonths, func(i, j int) bool {
		return monthOrder[emptyMonths[i]] < monthOrder[emptyMonths[j]]
	})

	// Логируем результаты
	if len(successfulMonths) > 0 {
		u.logger.Info("Successful cache updates for months", zap.Strings("months", successfulMonths))
	} else {
		u.logger.Warn("No successful cache updates for any months")
	}
	if len(emptyMonths) > 0 {
		u.logger.Info("No releases found for months", zap.Strings("months", emptyMonths))
	} else {
		u.logger.Info("Releases found for all months")
	}
	if totalReleases == 0 {
		u.logger.Warn("Cache initialization completed, but no releases were added")
	} else {
		u.logger.Info("Cache updated successfully", zap.Int("total_releases", totalReleases))
	}

	// Устанавливаем статус завершения обновления
	if u.metrics != nil {
		u.metrics.SetCacheUpdateStatus(false)
		// Устанавливаем время следующего обновления после завершения текущего
		nextUpdate := time.Now().Add(u.config.CacheDuration)
		u.metrics.SetNextCacheUpdate(nextUpdate)
	}

	if len(errs) > 0 {
		return fmt.Errorf("cache initialization completed with %d errors: %v", len(errs), errs)
	}

	return nil
}

// processMonth processes a single month
func (u *Impl) processMonth(ctx context.Context, month string, monthlyLinks []string, totalReleases *int, totalReleasesMu *sync.Mutex, successfulMonths, emptyMonths *[]string, monthsMu *sync.Mutex) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	startTime := time.Now()

	fullWhitelist := make([]string, 0, len(u.artistList.GetUnitedWhitelist()))
	whitelistMap := make(map[string]struct{})
	for _, artist := range u.artistList.GetUnitedWhitelist() {
		fullWhitelist = append(fullWhitelist, artist)
		whitelistMap[artist] = struct{}{}
	}
	sort.Strings(fullWhitelist)
	u.logger.Debug("Whitelist for caching", zap.Strings("whitelist", fullWhitelist), zap.String("month", month))

	var allReleases []release.Release
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, link := range monthlyLinks {
		// Проверяем контекст перед запуском новой горутины
		select {
		case <-ctx.Done():
			u.logger.Debug("Context cancelled, stopping processMonth early", zap.String("month", month), zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		link := link // capture range variable
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Проверяем контекст в начале горутины
			select {
			case <-ctx.Done():
				u.logger.Debug("Context cancelled, skipping page processing", zap.String("url", link))
				return
			default:
			}

			err := middleware.WithRetries(u.config.MaxRetries, u.config.RequestDelay, u.logger)(
				ctx, u.logger,
				func() error {
					rels, err := u.scraper.ParseMonthlyPage(ctx, link, month, whitelistMap)
					if err != nil {
						if ctx.Err() != nil {
							u.logger.Debug("ParseMonthlyPage cancelled due to context cancellation", zap.String("url", link), zap.Error(ctx.Err()))
							return ctx.Err()
						}
						u.logger.Error("Failed to parse page", zap.String("url", link), zap.Error(err))
						return err
					}
					if len(rels) > 0 {
						mu.Lock()
						totalReleasesMu.Lock()
						*totalReleases += len(rels)
						totalReleasesMu.Unlock()
						allReleases = append(allReleases, rels...)
						mu.Unlock()
					}
					return nil
				},
			)
			if err != nil && ctx.Err() == nil {
				u.logger.Error("Failed to process page", zap.String("url", link), zap.Error(err))
			}
		}()
	}

	// Ждем завершения всех горутин с возможностью отмены
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		u.logger.Debug("Context cancelled while waiting for month processing", zap.String("month", month), zap.Error(ctx.Err()))
		return ctx.Err()
	case <-done:
		// Продолжаем обработку результатов
	}

	duration := time.Since(startTime)
	if len(allReleases) > 0 {
		u.cache.StoreReleases(month, allReleases)
		monthsMu.Lock()
		*successfulMonths = append(*successfulMonths, month)
		monthsMu.Unlock()
		u.logger.Info("Cached releases for month", zap.String("month", month), zap.Int("release_count", len(allReleases)), zap.String("duration", formatDuration(duration)))
	} else {
		monthsMu.Lock()
		*emptyMonths = append(*emptyMonths, month)
		monthsMu.Unlock()
		u.logger.Debug("No releases found for month", zap.String("month", month), zap.String("duration", formatDuration(duration)))
	}

	return nil
}

// StartUpdater periodically updates the cache
func (u *Impl) StartUpdater(ctx context.Context) {
	u.logger.Info("Starting cache updater", zap.Duration("cache_duration", u.config.CacheDuration))
	ticker := time.NewTicker(u.config.CacheDuration)
	defer ticker.Stop()

	// Немедленное синхронное обновление кэша при старте
	u.logger.Info("Starting initial cache update")
	if err := u.InitializeCache(ctx); err != nil {
		u.logger.Error("Initial cache update failed", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			u.logger.Info("Cache updater stopped due to context cancellation")
			return
		case t := <-ticker.C:
			u.logger.Info("Starting periodic cache update", zap.Time("tick_time", t))

			// Проверяем, что контекст еще не отменен перед запуском обновления
			select {
			case <-ctx.Done():
				u.logger.Info("Cache updater stopped during periodic update due to context cancellation")
				return
			default:
				// Создаем контекст с таймаутом для этого обновления
				updateCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
				go func() {
					defer cancel()
					if err := u.InitializeCache(updateCtx); err != nil {
						u.logger.Error("Periodic cache update failed", zap.Error(err))
					}
				}()
			}
		}
	}
}
