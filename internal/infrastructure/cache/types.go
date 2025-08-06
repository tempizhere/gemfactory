// Package cache содержит типы для кэширования релизов.
package cache

import (
	"context"
	"gemfactory/internal/config"
	"gemfactory/internal/domain/artist"
	"gemfactory/internal/domain/release"
	"gemfactory/internal/gateway/scraper"
	"gemfactory/internal/infrastructure/metrics"
	"gemfactory/internal/infrastructure/worker"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Cache defines the interface for cache operations
type Cache interface {
	GetReleasesForMonths(months []string, whitelist map[string]struct{}, femaleOnly, maleOnly bool) ([]release.Release, []string, error)
	ScheduleUpdate(ctx context.Context)
	Clear()
	StartUpdater(ctx context.Context)
	GetCachedLinks(month string) ([]string, error)
	IsUpdating(month string) bool
	StoreReleases(month string, releases []release.Release)
	GetCachedReleasesCount() int
}

// Entry holds cached releases or links
type Entry struct {
	Releases  []release.Release
	Links     []string
	Timestamp time.Time
}

// Updater defines the interface for cache updating
type Updater interface {
	InitializeCache(ctx context.Context) error
	StartUpdater(ctx context.Context)
}

// Manager manages the cache
type Manager struct {
	cache          map[string]Entry
	mu             sync.Mutex
	duration       time.Duration
	isUpdating     bool
	pendingUpdates map[string]struct{}
	logger         *zap.Logger
	config         *config.Config
	artistList     artist.WhitelistManager
	scraper        scraper.Fetcher
	updater        Updater
	workerPool     worker.PoolInterface
	metrics        metrics.Interface
	// Поля для дебаунса обновлений
	lastUpdateTime time.Time
	updateDebounce *time.Timer
}

var _ Cache = (*Manager)(nil)

// NewManager создает новый экземпляр Manager
func NewManager(config *config.Config, logger *zap.Logger, al artist.WhitelistManager, scraper scraper.Fetcher, updater Updater, workerPool worker.PoolInterface) *Manager {
	cacheDuration := parseCacheDuration(logger, config)

	// Используем переданный worker pool вместо создания собственного
	// workerPool := worker.NewWorkerPool(config.MaxConcurrentRequests, 50, logger)

	return &Manager{
		cache:          make(map[string]Entry),
		duration:       cacheDuration,
		logger:         logger,
		config:         config,
		artistList:     al,
		scraper:        scraper,
		updater:        updater,
		isUpdating:     false,
		pendingUpdates: make(map[string]struct{}),
		workerPool:     workerPool, // Используем переданный worker pool
	}
}

// SetUpdater sets the updater for the Manager
func (cm *Manager) SetUpdater(updater Updater) {
	cm.updater = updater
}

// SetMetrics sets the metrics interface for the Manager
func (cm *Manager) SetMetrics(metrics metrics.Interface) {
	cm.metrics = metrics
}

// parseCacheDuration parses the CACHE_DURATION environment variable
func parseCacheDuration(logger *zap.Logger, config *config.Config) time.Duration {
	cacheDuration := config.CacheDuration
	if cacheDuration <= 0 {
		logger.Warn("Invalid CACHE_DURATION, using default", zap.Duration("default", 8*time.Hour))
		return 8 * time.Hour
	}
	logger.Info("CACHE_DURATION parsed successfully", zap.Duration("duration", cacheDuration))
	return cacheDuration
}
