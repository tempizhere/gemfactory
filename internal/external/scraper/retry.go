// Package scraper содержит retry логику для веб-скрапинга.
package scraper

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
)

// WithRetry выполняет функцию с retry логикой
func WithRetry(ctx context.Context, logger *zap.Logger, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Проверяем контекст перед каждой попыткой
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Выполняем функцию
		err := fn()
		if err == nil {
			if attempt > 0 {
				logger.Debug("Function succeeded after retry",
					zap.Int("attempt", attempt+1),
					zap.Int("max_retries", config.MaxRetries))
			}
			return nil
		}

		lastErr = err

		if attempt == config.MaxRetries {
			break
		}

		// Вычисляем задержку с экспоненциальным backoff
		delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffMultiplier, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		logger.Debug("Function failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", config.MaxRetries),
			zap.Duration("delay", delay),
			zap.Error(err))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Продолжаем к следующей попытке
		}
	}

	return fmt.Errorf("function failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}
