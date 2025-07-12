// Package updater содержит типы для обновления релизов.
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

// Impl implements the Updater interface
type Impl struct {
	config     *config.Config
	logger     *zap.Logger
	artistList artist.WhitelistManager
	cache      cache.Cache
	scraper    scraper.Fetcher
}

// NewUpdater creates a new Updater instance
func NewUpdater(config *config.Config, logger *zap.Logger, al artist.WhitelistManager, cache cache.Cache, scraper scraper.Fetcher) *Impl {
	return &Impl{
		config:     config,
		logger:     logger,
		artistList: al,
		cache:      cache,
		scraper:    scraper,
	}
}
