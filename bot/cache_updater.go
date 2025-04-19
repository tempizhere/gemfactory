package bot

import (
	"os"
	"time"

	"gemfactory/parser"

	"go.uber.org/zap"
)

// StartCacheUpdater runs a background task to periodically update the cache based on CACHE_DURATION
func StartCacheUpdater(logger *zap.Logger) {
	// Чтение CACHE_DURATION из .env
	cacheDurationStr := os.Getenv("CACHE_DURATION")
	cacheDuration, err := time.ParseDuration(cacheDurationStr)
	if err != nil || cacheDuration <= 0 {
		cacheDuration = 24 * time.Hour // Значение по умолчанию
		logger.Warn("Invalid CACHE_DURATION, using default", zap.String("cache_duration", cacheDurationStr), zap.Duration("default", cacheDuration))
	}

	logger.Info("Starting cache updater", zap.Duration("interval", cacheDuration))

	for {
		// Ждём истечения CACHE_DURATION
		time.Sleep(cacheDuration)

		// Обновляем кэш
		logger.Info("Cache update started")
		parser.UpdateCache(logger)
		logger.Info("Cache update completed")
	}
}