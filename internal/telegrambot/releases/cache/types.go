package cache

import (
	"context"
	"sync"
	"time"

	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/pkg/config"
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
}

// CacheEntry holds cached releases or links
type CacheEntry struct {
	Releases  []release.Release
	Links     []string
	Timestamp time.Time
}

// CacheManager manages the cache
type CacheManager struct {
	cache                    map[string]CacheEntry
	mu                       sync.RWMutex
	duration                 time.Duration
	updateTimer              *time.Timer
	updateTimerMu            sync.Mutex
	isUpdating               bool
	pendingUpdates           map[string]struct{}
	pendingUpdatesTimestamps map[string]time.Time
	logger                   *zap.Logger
	config                   *config.Config
	artistList               *artistlist.ArtistList
	scraper                  Scraper
	updater                  Updater
}

// Scraper defines the interface for scraping operations
type Scraper interface {
	GetMonthlyLinksWithContext(ctx context.Context, months []string, config *config.Config, logger *zap.Logger) ([]string, error)
	ParseMonthlyPageWithContext(ctx context.Context, url string, whitelist map[string]struct{}, month string, config *config.Config, logger *zap.Logger) ([]release.Release, error)
}

// Updater defines the interface for cache updating
type Updater interface {
	InitializeCache(ctx context.Context) error
	StartUpdater()
}

// NewCacheManager creates a new CacheManager instance
func NewCacheManager(config *config.Config, logger *zap.Logger, al *artistlist.ArtistList, scraper Scraper, updater Updater) *CacheManager {
	cacheDuration := parseCacheDuration(logger, config)
	return &CacheManager{
		cache:                    make(map[string]CacheEntry),
		duration:                 cacheDuration,
		logger:                   logger,
		config:                   config,
		artistList:               al,
		scraper:                  scraper,
		updater:                  updater,
		isUpdating:               false,
		pendingUpdates:           make(map[string]struct{}),
		pendingUpdatesTimestamps: make(map[string]time.Time),
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
