package cache

import (
	"context"
	"gemfactory/internal/telegrambot/releases/artist"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/scraper"
	"gemfactory/pkg/config"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Cache defines the interface for cache operations
type Cache interface {
	GetReleasesForMonths(months []string, whitelist map[string]struct{}, femaleOnly, maleOnly bool) ([]release.Release, []string, error)
	ScheduleUpdate()
	Clear()
	StartUpdater()
	GetCachedLinks(month string) ([]string, error)
	IsUpdating(month string) bool
	StoreReleases(month string, releases []release.Release)
}

// CacheEntry holds cached releases or links
type CacheEntry struct {
	Releases  []release.Release
	Links     []string
	Timestamp time.Time
}

// Updater defines the interface for cache updating
type Updater interface {
	InitializeCache(ctx context.Context) error
	StartUpdater()
}

// CacheManager manages the cache
type CacheManager struct {
	cache          map[string]CacheEntry
	mu             sync.Mutex
	duration       time.Duration
	isUpdating     bool
	pendingUpdates map[string]struct{}
	logger         *zap.Logger
	config         *config.Config
	artistList     artist.WhitelistManager
	scraper        scraper.Fetcher
	updater        Updater
}

// NewCacheManager creates a new CacheManager instance
func NewCacheManager(config *config.Config, logger *zap.Logger, al artist.WhitelistManager, scraper scraper.Fetcher, updater Updater) *CacheManager {
	cacheDuration := parseCacheDuration(logger, config)
	return &CacheManager{
		cache:          make(map[string]CacheEntry),
		duration:       cacheDuration,
		logger:         logger,
		config:         config,
		artistList:     al,
		scraper:        scraper,
		updater:        updater,
		isUpdating:     false,
		pendingUpdates: make(map[string]struct{}),
	}
}

// SetUpdater sets the updater for the CacheManager
func (cm *CacheManager) SetUpdater(updater Updater) {
	cm.updater = updater
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
