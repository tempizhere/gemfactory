package scraper

import (
	"context"

	"gemfactory/internal/telegrambot/releases/parser"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/pkg/config"
	"go.uber.org/zap"
)

// ScraperImpl implements the cache.Scraper interface
type ScraperImpl struct {
	config *config.Config
	logger *zap.Logger
}

// NewScraper creates a new ScraperImpl instance
func NewScraper(config *config.Config, logger *zap.Logger) *ScraperImpl {
	return &ScraperImpl{
		config: config,
		logger: logger,
	}
}

// GetMonthlyLinksWithContext retrieves links to monthly schedules with context
func (s *ScraperImpl) GetMonthlyLinksWithContext(ctx context.Context, months []string, config *config.Config, logger *zap.Logger) ([]string, error) {
	return GetMonthlyLinksWithContext(ctx, months, config, logger)
}

// ParseMonthlyPageWithContext parses a monthly schedule page with context
func (s *ScraperImpl) ParseMonthlyPageWithContext(ctx context.Context, url string, whitelist map[string]struct{}, month string, config *config.Config, logger *zap.Logger) ([]release.Release, error) {
	return parser.ParseMonthlyPageWithContext(ctx, url, whitelist, month, config, logger)
}
