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
			// TODO: Отправить сообщение об ошибке
			// if err := sendMessage(update.Message.Chat.ID, "❌ Невозможно определить пользователя"); err != nil {
			// 	logger.Error("Failed to send error message", zap.Error(err))
			// }
			return
		}

		if update.Message.From.UserName != adminUsername {
			user := getUserIdentifier(update.Message.From)
			logger.Warn("Unauthorized access attempt",
				zap.String("command", update.Message.Command()),
				zap.String("user", user),
				zap.String("expected_admin", adminUsername))

			// TODO: Отправить сообщение об отказе в доступе
			// if err := sendMessage(update.Message.Chat.ID, "🔒 Эта команда доступна только администратору"); err != nil {
			// 	logger.Error("Failed to send access denied message", zap.Error(err))
			// }
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
			// TODO: Отправить сообщение об ошибке
			// if err := sendMessage(update.Message.Chat.ID, "❌ Невозможно определить пользователя"); err != nil {
			// 	logger.Error("Failed to send error message", zap.Error(err))
			// }
			return nil
		}

		if update.Message.From.UserName != adminUsername {
			user := getUserIdentifier(update.Message.From)
			logger.Warn("Unauthorized access attempt",
				zap.String("command", update.Message.Command()),
				zap.String("user", user),
				zap.String("expected_admin", adminUsername))

			// TODO: Отправить сообщение об отказе в доступе
			// if err := sendMessage(update.Message.Chat.ID, "🔒 Эта команда доступна только администратору"); err != nil {
			// 	logger.Error("Failed to send access denied message", zap.Error(err))
			// }
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
