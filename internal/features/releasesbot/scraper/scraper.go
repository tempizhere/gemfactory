package scraper

import (
	"context"
	"net/http"
	"time"

	"gemfactory/internal/features/releasesbot/release"
	"gemfactory/pkg/config"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"go.uber.org/zap"
)

// NewCollector creates a new Colly collector with configured settings
func NewCollector(config *config.Config, logger *zap.Logger) *colly.Collector {
	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Устанавливаем HTTP-клиент с таймаутом
	collector.WithTransport(&http.Transport{
		ResponseHeaderTimeout: 60 * time.Second,
		DisableKeepAlives:     true,
	})

	if err := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       config.RequestDelay * 2, // Задержка 6s при REQUEST_DELAY=3s
		RandomDelay: config.RequestDelay,
	}); err != nil {
		logger.Error("Failed to set collector limit", zap.Error(err))
	}

	// Реализуем повторы вручную через OnError
	collector.OnError(func(r *colly.Response, err error) {
		maxRetries := config.MaxRetries
		retries := r.Request.Ctx.GetAny("retries")
		retryCount, ok := retries.(int)
		if !ok {
			retryCount = 0
		}

		if retryCount < maxRetries {
			retryCount++
			r.Request.Ctx.Put("retries", retryCount)
			logger.Warn("Retrying request", zap.String("url", r.Request.URL.String()), zap.Int("retry", retryCount), zap.Error(err))
			if err := r.Request.Retry(); err != nil {
				logger.Error("Failed to retry request", zap.String("url", r.Request.URL.String()), zap.Int("retry", retryCount), zap.Error(err))
			}
			return
		}

		logger.Error("Request failed after max retries", zap.String("url", r.Request.URL.String()), zap.Int("retries", retryCount), zap.Error(err))
	})

	// Добавляем случайный User-Agent
	extensions.RandomUserAgent(collector)

	return collector
}

// GetMonthlyLinks retrieves links to monthly schedules
func GetMonthlyLinks(months []string, config *config.Config, logger *zap.Logger) ([]string, error) {
	return GetMonthlyLinksWithContext(context.Background(), months, config, logger)
}

// ParseMonthlyPage parses a monthly schedule page
func ParseMonthlyPage(url string, whitelist map[string]struct{}, targetMonth string, config *config.Config, logger *zap.Logger) ([]release.Release, error) {
	return ParseMonthlyPageWithContext(context.Background(), url, whitelist, targetMonth, config, logger)
}
