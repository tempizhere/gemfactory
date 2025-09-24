// Package middleware содержит middleware для recovery и обработки ошибок.
package middleware

import (
	"runtime/debug"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// RecoveryMiddleware обрабатывает панику
func RecoveryMiddleware(logger *zap.Logger) func(next func()) {
	return func(next func()) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic recovered",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())))
			}
		}()
		next()
	}
}

// RecoveryMiddlewareWithUpdate обрабатывает панику с контекстом обновления
func RecoveryMiddlewareWithUpdate(logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update)) {
	return func(update tgbotapi.Update, next func(tgbotapi.Update)) {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				if update.Message != nil {
					user := getUserIdentifier(update.Message.From)
					logger.Error("Panic recovered in recovery middleware",
						zap.String("command", update.Message.Command()),
						zap.Int64("chat_id", update.Message.Chat.ID),
						zap.String("user", user),
						zap.Int("update_id", update.UpdateID),
						zap.Any("panic", panicErr),
						zap.String("stack", string(debug.Stack())))

					// TODO: Отправить пользователю сообщение об ошибке
					// if err := sendMessage(update.Message.Chat.ID, "❌ Произошла серьезная ошибка. Попробуйте позже."); err != nil {
					// 	logger.Error("Failed to send panic message", zap.Error(err))
					// }
				} else {
					logger.Error("Panic recovered in recovery middleware",
						zap.Int("update_id", update.UpdateID),
						zap.Any("panic", panicErr),
						zap.String("stack", string(debug.Stack())))
				}
			}
		}()
		next(update)
	}
}

// ErrorHandlerMiddleware обрабатывает ошибки от обработчиков с лучшим контекстом
func ErrorHandlerMiddleware(logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
	return func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				if update.Message != nil {
					user := getUserIdentifier(update.Message.From)
					logger.Error("Panic recovered in error handler",
						zap.String("command", update.Message.Command()),
						zap.Int64("chat_id", update.Message.Chat.ID),
						zap.String("user", user),
						zap.Int("update_id", update.UpdateID),
						zap.Any("panic", panicErr),
						zap.String("stack", string(debug.Stack())))

					// TODO: Отправить пользователю сообщение об ошибке
					// if err := sendMessage(update.Message.Chat.ID, "❌ Произошла серьезная ошибка. Попробуйте позже."); err != nil {
					// 	logger.Error("Failed to send panic message", zap.Error(err))
					// }
				} else {
					logger.Error("Panic recovered in error handler",
						zap.Int("update_id", update.UpdateID),
						zap.Any("panic", panicErr),
						zap.String("stack", string(debug.Stack())))
				}
			}
		}()

		err := next(update)
		if err != nil && update.Message != nil {
			user := getUserIdentifier(update.Message.From)

			// Определяем тип ошибки для лучшего логирования
			switch {
			case isCommandError(err):
				logger.Warn("Command error",
					zap.String("command", update.Message.Command()),
					zap.Int64("chat_id", update.Message.Chat.ID),
					zap.String("user", user),
					zap.Int("update_id", update.UpdateID),
					zap.Error(err))

			case isBotError(err):
				logger.Error("Bot error",
					zap.String("command", update.Message.Command()),
					zap.Int64("chat_id", update.Message.Chat.ID),
					zap.String("user", user),
					zap.Int("update_id", update.UpdateID),
					zap.Error(err))

			default:
				logger.Error("Unknown error",
					zap.String("command", update.Message.Command()),
					zap.Int64("chat_id", update.Message.Chat.ID),
					zap.String("user", user),
					zap.Int("update_id", update.UpdateID),
					zap.Error(err))
			}

			// TODO: Отправить пользователю информативное сообщение об ошибке
			// if sendErr := sendMessage(update.Message.Chat.ID, errorMessage); sendErr != nil {
			// 	logger.Error("Failed to send error message", zap.Error(sendErr))
			// }
		}

		return err
	}
}

// isCommandError проверяет, является ли ошибка ошибкой команды
func isCommandError(err error) bool {
	if err == nil {
		return false
	}
	// Простая проверка по тексту ошибки
	errorText := strings.ToLower(err.Error())
	return strings.Contains(errorText, "command") ||
		strings.Contains(errorText, "usage") ||
		strings.Contains(errorText, "invalid")
}

// isBotError проверяет, является ли ошибка внутренней ошибкой бота
func isBotError(err error) bool {
	if err == nil {
		return false
	}
	// Простая проверка по тексту ошибки
	errorText := strings.ToLower(err.Error())
	return strings.Contains(errorText, "internal") ||
		strings.Contains(errorText, "database") ||
		strings.Contains(errorText, "service")
}
