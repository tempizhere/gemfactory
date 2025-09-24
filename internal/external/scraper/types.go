// Package scraper содержит типы для веб-скрапинга.
package scraper

import (
	"context"
	"gemfactory/internal/model"
	"time"
)

// Fetcher определяет интерфейс для получения данных о релизах
type Fetcher interface {
	FetchMonthlyLinks(ctx context.Context, months []string, year string) ([]string, error)
	ParseMonthlyPage(ctx context.Context, url, month, year string, artists map[string]bool) ([]Release, error)
}

// Config представляет конфигурацию скрейпера
type Config struct {
	HTTPClientConfig HTTPClientConfig
	RetryConfig      RetryConfig
	RequestDelay     time.Duration
	LLMConfig        LLMConfig
}

// LLMConfig представляет конфигурацию LLM клиента
type LLMConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// HTTPClientConfig представляет конфигурацию HTTP клиента
type HTTPClientConfig struct {
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	DisableKeepAlives     bool
}

// RetryConfig представляет конфигурацию retry механизма
type RetryConfig struct {
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// Release представляет релиз (временный тип для парсера)
type Release struct {
	Date       string
	TimeMSK    string
	Artist     string
	AlbumName  string
	TitleTrack string
	MV         string
}

// ToModelRelease конвертирует scraper.Release в model.Release
// Внимание: требует дополнительной обработки для получения artist_id и release_type_id
func (r *Release) ToModelRelease() *model.Release {
	return &model.Release{
		Title:      r.AlbumName, // Используем AlbumName как Title
		AlbumName:  r.AlbumName,
		TitleTrack: r.TitleTrack,
		Date:       r.Date,
		TimeMSK:    r.TimeMSK,
		MV:         r.MV,
	}
}

// ScrapedRelease представляет релиз, полученный при скрапинге
type ScrapedRelease struct {
	Artist    string    `json:"artist"`
	Title     string    `json:"title"`
	Date      string    `json:"date"`
	Type      string    `json:"type"`
	Gender    string    `json:"gender"`
	Month     string    `json:"month"`
	Year      int       `json:"year"`
	ScrapedAt time.Time `json:"scraped_at"`
}

// ScrapingResult представляет результат скрапинга
type ScrapingResult struct {
	Releases   []ScrapedRelease `json:"releases"`
	TotalCount int              `json:"total_count"`
	Month      string           `json:"month"`
	ScrapedAt  time.Time        `json:"scraped_at"`
	Error      string           `json:"error,omitempty"`
}

// ScrapingConfig представляет конфигурацию скрапинга
type ScrapingConfig struct {
	BaseURL     string        `json:"base_url"`
	Delay       time.Duration `json:"delay"`
	Timeout     time.Duration `json:"timeout"`
	RetryCount  int           `json:"retry_count"`
	UserAgent   string        `json:"user_agent"`
	MaxReleases int           `json:"max_releases"`
}

// ScrapingStats представляет статистику скрапинга
type ScrapingStats struct {
	TotalRequests      int           `json:"total_requests"`
	SuccessfulRequests int           `json:"successful_requests"`
	FailedRequests     int           `json:"failed_requests"`
	TotalReleases      int           `json:"total_releases"`
	ScrapingTime       time.Duration `json:"scraping_time"`
	LastScraped        time.Time     `json:"last_scraped"`
}

// ArtistBlock представляет блок HTML с артистом
type ArtistBlock struct {
	HTML   string
	Artist string
	Row    int
}
