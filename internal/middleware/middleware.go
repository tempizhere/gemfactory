// Package middleware содержит middleware компоненты.
package middleware

import (
	"gemfactory/internal/config"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Middleware представляет middleware компонент
type Middleware struct {
	rateLimiter RateLimiterInterface
	debouncer   DebouncerInterface
	logger      *zap.Logger
	config      *config.Config
}

// New создает новый middleware
func New(config *config.Config, logger *zap.Logger) *Middleware {
	// Создаем rate limiter (10 запросов в минуту)
	rateLimiter := NewRateLimiter(10, 60*time.Second, logger)

	// Создаем debouncer (1 секунда между запросами)
	debouncer := NewDebouncer(1*time.Second, logger)

	return &Middleware{
		rateLimiter: rateLimiter,
		debouncer:   debouncer,
		logger:      logger,
		config:      config,
	}
}

// Process обрабатывает обновление через middleware
func (m *Middleware) Process(update tgbotapi.Update) bool {
	// Применяем rate limiting
	if update.Message != nil {
		userID := update.Message.From.ID
		if !m.rateLimiter.Allow(userID) {
			m.logger.Warn("Rate limit exceeded", zap.Int64("user_id", userID))
			return false
		}
	}

	return true
}

// ProcessWithMiddleware применяет все middleware к обновлению
func (m *Middleware) ProcessWithMiddleware(update tgbotapi.Update, handler func(tgbotapi.Update)) {
	// Создаем цепочку middleware
	middlewareChain := func(update tgbotapi.Update) {
		// Recovery middleware
		RecoveryMiddlewareWithUpdate(m.logger)(update, func(update tgbotapi.Update) {
			// Logging middleware
			LoggingMiddleware(m.logger)(update, func(update tgbotapi.Update) {
				// Debounce middleware for messages
				DebounceMiddleware(m.debouncer, m.logger)(update, func(update tgbotapi.Update) {
					// Debounce middleware for callbacks
					DebounceCallbackMiddleware(m.debouncer, m.logger)(update, func(update tgbotapi.Update) {
						// Rate limiting
						if m.Process(update) {
							handler(update)
						}
					})
				})
			})
		})
	}

	middlewareChain(update)
}

// ProcessWithMiddlewareAndError применяет все middleware к обновлению с обработкой ошибок
func (m *Middleware) ProcessWithMiddlewareAndError(update tgbotapi.Update, handler func(tgbotapi.Update) error) error {
	// Создаем цепочку middleware с обработкой ошибок
	middlewareChain := func(update tgbotapi.Update) error {
		// Error handler middleware
		return ErrorHandlerMiddleware(m.logger)(update, func(update tgbotapi.Update) error {
			// Logging middleware with error
			return LogRequestWithError(m.logger)(update, func(update tgbotapi.Update) error {
				// Debounce middleware with error
				return DebounceMiddlewareWithError(m.debouncer, m.logger)(update, func(update tgbotapi.Update) error {
					// Rate limiting
					if !m.Process(update) {
						return nil // Rate limit exceeded, skip handler
					}
					return handler(update)
				})
			})
		})
	}

	return middlewareChain(update)
}

// GetAdminMiddleware возвращает middleware для проверки прав администратора
func (m *Middleware) GetAdminMiddleware() func(update tgbotapi.Update, next func(tgbotapi.Update)) {
	return AdminOnlyMiddlewareWithConfig(m.config, m.logger)
}

// GetAdminMiddlewareWithError возвращает middleware для проверки прав администратора с обработкой ошибок
func (m *Middleware) GetAdminMiddlewareWithError() func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
	return AdminOnlyMiddlewareWithConfigAndError(m.config, m.logger)
}

// Cleanup очищает устаревшие записи в middleware
func (m *Middleware) Cleanup() {
	m.rateLimiter.Cleanup()
	m.debouncer.Cleanup()
}

// Register регистрирует middleware
func Register(config *config.Config, logger *zap.Logger) *Middleware {
	middleware := New(config, logger)
	logger.Info("Middleware registered")
	return middleware
}
