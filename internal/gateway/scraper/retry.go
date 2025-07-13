package scraper

import (
	"context"
	"math"
	"time"

	"go.uber.org/zap"
)

// RetryConfig конфигурация для retry механизма
type RetryConfig struct {
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// RetryableFunc функция, которая может быть повторена
type RetryableFunc func() error

// WithRetry выполняет функцию с retry механизмом
func WithRetry(ctx context.Context, logger *zap.Logger, config RetryConfig, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		if attempt == config.MaxRetries {
			break
		}

		delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffMultiplier, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		logger.Warn("Retry attempt failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", config.MaxRetries),
			zap.Duration("delay", delay),
			zap.Error(lastErr))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}
