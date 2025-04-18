package parser

import (
	"sync"
	"time"

	"github.com/gocolly/colly"
	"go.uber.org/zap"
)

// Collector is an interface for colly.Collector
type Collector interface {
	Visit(url string) error
	OnHTML(selector string, fn func(*colly.HTMLElement))
	OnResponse(fn func(*colly.Response))
	Wait()
}

// CollyCollector wraps colly.Collector
type CollyCollector struct {
	*colly.Collector
}

// OnHTML implements Collector interface
func (c *CollyCollector) OnHTML(selector string, fn func(*colly.HTMLElement)) {
	c.Collector.OnHTML(selector, fn)
}

// OnResponse implements Collector interface
func (c *CollyCollector) OnResponse(fn func(*colly.Response)) {
	c.Collector.OnResponse(fn)
}

// Wait implements Collector interface
func (c *CollyCollector) Wait() {
	c.Collector.Wait()
}

// NewCollector creates a new collector with common settings
func NewCollector(maxRetries int, delay time.Duration, logger *zap.Logger) Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*kpopofficial.com*", Delay: delay})

	// Увеличение таймаута
	c.SetRequestTimeout(60 * time.Second)

	// Настройка заголовков для имитации браузера
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Connection", "keep-alive")
	})

	retryCounts := make(map[string]int)
	var retryMu sync.Mutex

	c.OnError(func(r *colly.Response, err error) {
		url := r.Request.URL.String()
		retryMu.Lock()
		count := retryCounts[url]
		count++
		retryCounts[url] = count
		retryMu.Unlock()

		logger.Error("Request failed",
			zap.String("url", url),
			zap.Error(err),
			zap.Int("attempt", count),
			zap.Int("max_retries", maxRetries))

		if count < maxRetries {
			logger.Warn("Retrying request",
				zap.String("url", url),
				zap.Int("attempt", count))
			time.Sleep(10 * time.Second)
			_ = r.Request.Retry()
		}
	})

	return &CollyCollector{c}
}
