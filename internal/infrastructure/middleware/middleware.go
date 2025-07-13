// Package middleware реализует middleware для парсинга и обновления релизов.
package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// WithLogging logs HTTP requests and tasks
func WithLogging(_ *zap.Logger) Func {
	return func(r *colly.Request, l *zap.Logger) error {
		l.Debug("Visiting URL", zap.String("url", r.URL.String()))
		return nil
	}
}

// WithTaskLogging logs asynchronous tasks
func WithTaskLogging(_ *zap.Logger, taskName string) TaskMiddlewareFunc {
	return func(_ context.Context, l *zap.Logger, next func() error) error {
		l.Debug("Starting task", zap.String("task", taskName))
		err := next()
		if err != nil {
			l.Error("Task failed", zap.String("task", taskName), zap.Error(err))
			return err
		}
		l.Debug("Completed task", zap.String("task", taskName))
		return nil
	}
}

// WithRetries retries a task or request on failure
func WithRetries(maxRetries int, delay time.Duration, _ *zap.Logger) TaskMiddlewareFunc {
	return func(ctx context.Context, l *zap.Logger, next func() error) error {
		for attempt := 1; attempt <= maxRetries; attempt++ {
			select {
			case <-ctx.Done():
				return fmt.Errorf("task cancelled: %w", ctx.Err())
			default:
				if err := next(); err != nil {
					l.Warn("Retry attempt", zap.Int("attempt", attempt), zap.Error(err))
					if attempt < maxRetries {
						time.Sleep(delay * time.Duration(attempt))
						continue
					}
					return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
				}
				return nil
			}
		}
		return nil
	}
}
