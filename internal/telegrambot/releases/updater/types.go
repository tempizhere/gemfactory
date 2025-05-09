package updater

import (
	"context"
	"gemfactory/internal/telegrambot/releases/artist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/scraper"
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
	artistList artist.WhitelistManager
	cache      cache.Cache
	scraper    scraper.Fetcher
}

// NewUpdater creates a new Updater instance
func NewUpdater(config *config.Config, logger *zap.Logger, al artist.WhitelistManager, cache cache.Cache, scraper scraper.Fetcher) *UpdaterImpl {
	return &UpdaterImpl{
		config:     config,
		logger:     logger,
		artistList: al,
		cache:      cache,
		scraper:    scraper,
	}
}
