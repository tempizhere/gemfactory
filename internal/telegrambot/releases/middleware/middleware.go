package middleware

import (
	"context"
	"fmt"
	"gemfactory/pkg/config"
	"time"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// WithLogging logs HTTP requests and tasks
func WithLogging(logger *zap.Logger) MiddlewareFunc {
	return func(r *colly.Request, l *zap.Logger) error {
		l.Debug("Visiting URL", zap.String("url", r.URL.String()))
		return nil
	}
}

// WithTaskLogging logs asynchronous tasks
func WithTaskLogging(logger *zap.Logger, taskName string) TaskMiddlewareFunc {
	return func(ctx context.Context, l *zap.Logger, next func() error) error {
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
func WithRetries(maxRetries int, delay time.Duration, logger *zap.Logger) TaskMiddlewareFunc {
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

// WithQueue limits concurrent tasks or requests
func WithQueue(config *config.Config, logger *zap.Logger) TaskMiddlewareFunc {
	type task struct {
		ctx  context.Context
		fn   func() error
		done chan error
	}

	queue := make(chan task, config.MaxConcurrentRequests)
	worker := func(ctx context.Context) {
		for t := range queue {
			select {
			case <-ctx.Done():
				t.done <- ctx.Err()
				continue
			case <-t.ctx.Done():
				t.done <- t.ctx.Err()
				continue
			default:
				err := t.fn()
				t.done <- err
			}
		}
	}

	// Start workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < config.MaxConcurrentRequests; i++ {
		go worker(ctx)
	}

	return func(ctx context.Context, l *zap.Logger, next func() error) error {
		done := make(chan error, 1)
		select {
		case queue <- task{ctx: ctx, fn: next, done: done}:
			return <-done
		case <-ctx.Done():
			return fmt.Errorf("queue task cancelled: %w", ctx.Err())
		}
	}
}
