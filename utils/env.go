package utils

import (
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// GetCacheCheckInterval reads CACHE_CHECK_INTERVAL from .env
func GetCacheCheckInterval() time.Duration {
	cacheCheckIntervalStr := os.Getenv("CACHE_CHECK_INTERVAL")
	cacheCheckInterval, err := time.ParseDuration(cacheCheckIntervalStr)
	if err != nil {
		// Логируем ошибку и используем значение по умолчанию
		logger, _ := zap.NewProduction()
		logger.Warn("Invalid CACHE_CHECK_INTERVAL, using default of 30 minutes", zap.Error(err), zap.String("value", cacheCheckIntervalStr))
		return 30 * time.Minute
	}

	// Проверяем минимальный интервал (1 минута)
	if cacheCheckInterval < 1*time.Minute {
		logger, _ := zap.NewProduction()
		logger.Warn("CACHE_CHECK_INTERVAL is too small, using minimum of 1 minute", zap.Duration("value", cacheCheckInterval))
		return 1 * time.Minute
	}

	return cacheCheckInterval
}

// GetRequestDelay reads REQUEST_DELAY from .env
func GetRequestDelay() time.Duration {
	delayStr := os.Getenv("REQUEST_DELAY")
	delay, err := time.ParseDuration(delayStr)
	if err != nil || delay <= 0 {
		// Логируем ошибку и используем значение по умолчанию
		logger, _ := zap.NewProduction()
		logger.Warn("Invalid REQUEST_DELAY, using default of 10 seconds", zap.Error(err), zap.String("value", delayStr))
		return 10 * time.Second
	}
	return delay
}

// GetMaxRetries reads MAX_RETRIES from .env
func GetMaxRetries() int {
	maxRetriesStr := os.Getenv("MAX_RETRIES")
	maxRetries, err := strconv.Atoi(maxRetriesStr)
	if err != nil || maxRetries <= 0 {
		// Логируем ошибку и используем значение по умолчанию
		logger, _ := zap.NewProduction()
		logger.Warn("Invalid MAX_RETRIES, using default of 3", zap.Error(err), zap.String("value", maxRetriesStr))
		return 3
	}
	return maxRetries
}

// GetCollectorConfig reads MAX_RETRIES and REQUEST_DELAY from .env for collector configuration
func GetCollectorConfig() (maxRetries int, delay time.Duration) {
	maxRetriesStr := os.Getenv("MAX_RETRIES")
	maxRetries, err := strconv.Atoi(maxRetriesStr)
	if err != nil || maxRetries <= 0 {
		logger, _ := zap.NewProduction()
		logger.Warn("Invalid MAX_RETRIES, using default of 3", zap.Error(err), zap.String("value", maxRetriesStr))
		maxRetries = 3
	}

	delayStr := os.Getenv("REQUEST_DELAY")
	delay, err = time.ParseDuration(delayStr)
	if err != nil || delay <= 0 {
		logger, _ := zap.NewProduction()
		logger.Warn("Invalid REQUEST_DELAY, using default of 5 seconds", zap.Error(err), zap.String("value", delayStr))
		return maxRetries, 5 * time.Second
	}

	return maxRetries, delay
}
