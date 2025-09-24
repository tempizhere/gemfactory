package scraper

import (
	"context"
	"fmt"

	"gemfactory/internal/external/llm"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// fetcherImpl реализует интерфейс Fetcher
type fetcherImpl struct {
	config     Config
	logger     *zap.Logger
	httpClient *HTTPClient
	llmClient  *llm.Client
}

// NewFetcher создает новый экземпляр Fetcher
func NewFetcher(config Config, logger *zap.Logger) Fetcher {
	httpClient := NewHTTPClient(config.HTTPClientConfig, logger)
	llmClient := llm.NewClient(llm.Config{
		BaseURL: config.LLMConfig.BaseURL,
		APIKey:  config.LLMConfig.APIKey,
		Timeout: config.LLMConfig.Timeout,
	}, logger)

	return &fetcherImpl{
		config:     config,
		logger:     logger,
		httpClient: httpClient,
		llmClient:  llmClient,
	}
}

// FetchMonthlyLinks получает ссылки на страницы с расписанием релизов за указанные месяцы
func (f *fetcherImpl) FetchMonthlyLinks(ctx context.Context, months []string, year string) ([]string, error) {
	links := make([]string, 0, len(months))

	for _, month := range months {
		url := fmt.Sprintf("https://kpopofficial.com/kpop-comeback-schedule-%s-%s/", month, year)
		links = append(links, url)
		f.logger.Info("Generated monthly link", zap.String("url", url))
	}

	return links, nil
}

// newCollector creates a new Colly collector with configured middleware
func (f *fetcherImpl) newCollector() *colly.Collector {
	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Используем оптимизированный HTTP клиент
	if f.httpClient != nil {
		collector.WithTransport(f.httpClient.client.Transport)
	}

	// Настраиваем задержки
	_ = collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       f.config.RequestDelay,
	})

	// Добавляем middleware для логирования
	collector.OnRequest(func(r *colly.Request) {
		f.logger.Debug("Making request", zap.String("url", r.URL.String()))
	})

	collector.OnResponse(func(r *colly.Response) {
		f.logger.Debug("Received response",
			zap.String("url", r.Request.URL.String()),
			zap.Int("status", r.StatusCode),
			zap.Int("size", len(r.Body)))
	})

	return collector
}

// getMonthNumber возвращает номер месяца по его названию
func (f *fetcherImpl) getMonthNumber(month string) (string, bool) {
	months := map[string]string{
		"january":   "01",
		"february":  "02",
		"march":     "03",
		"april":     "04",
		"may":       "05",
		"june":      "06",
		"july":      "07",
		"august":    "08",
		"september": "09",
		"october":   "10",
		"november":  "11",
		"december":  "12",
	}

	monthNum, ok := months[month]
	return monthNum, ok
}
