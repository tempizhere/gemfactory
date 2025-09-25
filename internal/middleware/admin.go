// Package middleware содержит middleware для проверки прав администратора.
package middleware

import (
	"gemfactory/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// AdminOnlyMiddleware ограничивает доступ только администраторам с лучшей валидацией
func AdminOnlyMiddleware(adminUsername string, logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update)) {
	return func(update tgbotapi.Update, next func(tgbotapi.Update)) {
		if update.Message == nil {
			next(update)
			return
		}

		if update.Message.From == nil {
			logger.Warn("No user information in message")
			return
		}

		if update.Message.From.UserName != adminUsername {
			user := getUserIdentifier(update.Message.From)
			logger.Warn("Unauthorized access attempt",
				zap.String("command", update.Message.Command()),
				zap.String("user", user),
				zap.String("expected_admin", adminUsername))
			return
		}

		next(update)
	}
}

// AdminOnlyMiddlewareWithError ограничивает доступ только администраторам с обработкой ошибок
func AdminOnlyMiddlewareWithError(adminUsername string, logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
	return func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
		if update.Message == nil {
			return next(update)
		}

		if update.Message.From == nil {
			logger.Warn("No user information in message")
			return nil
		}

		if update.Message.From.UserName != adminUsername {
			user := getUserIdentifier(update.Message.From)
			logger.Warn("Unauthorized access attempt",
				zap.String("command", update.Message.Command()),
				zap.String("user", user),
				zap.String("expected_admin", adminUsername))
			return nil
		}

		return next(update)
	}
}

// AdminOnlyMiddlewareWithConfig ограничивает доступ только администраторам с использованием конфигурации
func AdminOnlyMiddlewareWithConfig(config *config.Config, logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update)) {
	return AdminOnlyMiddleware(config.AdminUsername, logger)
}

// AdminOnlyMiddlewareWithConfigAndError ограничивает доступ только администраторам с использованием конфигурации и обработкой ошибок
func AdminOnlyMiddlewareWithConfigAndError(config *config.Config, logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
	return AdminOnlyMiddlewareWithError(config.AdminUsername, logger)
}
