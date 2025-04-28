package updater

import (
	"context"

	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/pkg/config"
	"go.uber.org/zap"
)

// Updater defines the interface for cache updating
type Updater interface {
	InitializeCache(ctx context.Context) error
	StartUpdater()
}

// UpdaterImpl implements the Updater interface
type UpdaterImpl struct {
	config     *config.Config
	logger     *zap.Logger
	artistList *artistlist.ArtistList
	cache      Cache
	scraper    Scraper
}

// Cache defines the interface for cache operations
type Cache interface {
	GetCachedLinks(month string) ([]string, error)
	StoreReleases(month string, releases []release.Release)
}

// Scraper defines the interface for scraping operations
type Scraper interface {
	GetMonthlyLinksWithContext(ctx context.Context, months []string, config *config.Config, logger *zap.Logger) ([]string, error)
	ParseMonthlyPageWithContext(ctx context.Context, url string, whitelist map[string]struct{}, month string, config *config.Config, logger *zap.Logger) ([]release.Release, error)
}

// NewUpdater creates a new Updater instance
func NewUpdater(config *config.Config, logger *zap.Logger, al *artistlist.ArtistList, cache Cache, scraper Scraper) *UpdaterImpl {
	return &UpdaterImpl{
		config:     config,
		logger:     logger,
		artistList: al,
		cache:      cache,
		scraper:    scraper,
	}
}
