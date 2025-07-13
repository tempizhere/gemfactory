// Package updater содержит типы для обновления релизов.
package updater

import (
	"context"
	"gemfactory/internal/config"
	"gemfactory/internal/domain/artist"
	"gemfactory/internal/gateway/scraper"
	"gemfactory/internal/infrastructure/cache"
	"gemfactory/internal/infrastructure/metrics"

	"go.uber.org/zap"
)

// Updater defines the interface for cache updating
type Updater interface {
	InitializeCache(ctx context.Context) error
	StartUpdater(ctx context.Context)
	SetMetrics(metrics metrics.Interface)
}

// Impl implements the Updater interface
type Impl struct {
	config     *config.Config
	logger     *zap.Logger
	artistList artist.WhitelistManager
	cache      cache.Cache
	scraper    scraper.Fetcher
	metrics    metrics.Interface
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

// SetMetrics sets the metrics interface for the updater
func (u *Impl) SetMetrics(metrics metrics.Interface) {
	u.metrics = metrics
}
