// Package middleware содержит middleware для логирования запросов.
package middleware

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// RequestContext содержит контекст для обработки запроса
type RequestContext struct {
	StartTime time.Time
	RequestID string
	UserID    int64
	ChatID    int64
	Command   string
}

// LoggingMiddleware логирует входящие команды с контекстом
func LoggingMiddleware(logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update)) {
	return func(update tgbotapi.Update, next func(tgbotapi.Update)) {
		if update.Message == nil {
			next(update)
			return
		}

		requestCtx := &RequestContext{
			StartTime: time.Now(),
			RequestID: fmt.Sprintf("%d-%d", update.UpdateID, time.Now().UnixNano()),
			UserID:    update.Message.From.ID,
			ChatID:    update.Message.Chat.ID,
			Command:   update.Message.Command(),
		}

		user := getUserIdentifier(update.Message.From)

		logger.Info("Processing command",
			zap.String("request_id", requestCtx.RequestID),
			zap.String("command", requestCtx.Command),
			zap.Int64("user_id", requestCtx.UserID),
			zap.Int64("chat_id", requestCtx.ChatID),
			zap.String("user", user),
			zap.Int("update_id", update.UpdateID))

		// Выполняем следующий middleware/handler
		next(update)

		// Логируем завершение обработки
		duration := time.Since(requestCtx.StartTime)
		logger.Info("Command completed successfully",
			zap.String("request_id", requestCtx.RequestID),
			zap.String("command", requestCtx.Command),
			zap.Duration("duration", duration))
	}
}

// LogRequestWithError логирует запрос с обработкой ошибок
func LogRequestWithError(logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
	return func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
		if update.Message == nil {
			return next(update)
		}

		requestCtx := &RequestContext{
			StartTime: time.Now(),
			RequestID: fmt.Sprintf("%d-%d", update.UpdateID, time.Now().UnixNano()),
			UserID:    update.Message.From.ID,
			ChatID:    update.Message.Chat.ID,
			Command:   update.Message.Command(),
		}

		user := getUserIdentifier(update.Message.From)

		logger.Info("Processing command",
			zap.String("request_id", requestCtx.RequestID),
			zap.String("command", requestCtx.Command),
			zap.Int64("user_id", requestCtx.UserID),
			zap.Int64("chat_id", requestCtx.ChatID),
			zap.String("user", user),
			zap.Int("update_id", update.UpdateID))

		// Выполняем следующий middleware/handler
		err := next(update)

		// Логируем завершение обработки
		duration := time.Since(requestCtx.StartTime)
		if err != nil {
			logger.Error("Command completed with error",
				zap.String("request_id", requestCtx.RequestID),
				zap.String("command", requestCtx.Command),
				zap.Duration("duration", duration),
				zap.Error(err))
		} else {
			logger.Info("Command completed successfully",
				zap.String("request_id", requestCtx.RequestID),
				zap.String("command", requestCtx.Command),
				zap.Duration("duration", duration))
		}

		return err
	}
}

// getUserIdentifier возвращает идентификатор пользователя
func getUserIdentifier(user *tgbotapi.User) string {
	if user == nil {
		return "unknown"
	}

	if user.UserName != "" {
		return "@" + user.UserName
	}

	if user.FirstName != "" {
		if user.LastName != "" {
			return user.FirstName + " " + user.LastName
		}
		return user.FirstName
	}

	return fmt.Sprintf("user_%d", user.ID)
}
