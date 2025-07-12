// Package middleware содержит типы для middleware парсинга релизов.
package middleware

import (
	"context"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// Func defines a middleware function for HTTP requests
type Func func(*colly.Request, *zap.Logger) error

// TaskMiddlewareFunc defines a middleware function for asynchronous tasks
type TaskMiddlewareFunc func(context.Context, *zap.Logger, func() error) error
