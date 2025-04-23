package bot

import (
	"context"
	"time"

	"gemfactory/parser"

	"go.uber.org/zap"
)

// StartCacheUpdater starts a goroutine to update the cache every CACHE_DURATION
func StartCacheUpdater(ctx context.Context, logger *zap.Logger) {
	ticker := time.NewTicker(parser.GetCacheDuration())
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logger.Info("Triggering periodic cache update")
				parser.InitializeCache(logger)
				parser.CleanupOldCacheEntries()
			case <-ctx.Done():
				logger.Info("Stopping cache updater")
				return
			}
		}
	}()
}
