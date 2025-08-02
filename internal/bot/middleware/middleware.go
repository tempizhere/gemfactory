package middleware

import (
	"fmt"
	"gemfactory/internal/domain/types"
	"time"

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

// LogRequest logs incoming commands with context
func LogRequest(ctx types.Context, next types.HandlerFunc) error {
	requestCtx := &RequestContext{
		StartTime: time.Now(),
		RequestID: fmt.Sprintf("%d-%d", ctx.UpdateID, time.Now().UnixNano()),
		UserID:    ctx.Message.From.ID,
		ChatID:    ctx.Message.Chat.ID,
		Command:   ctx.Message.Command(),
	}

	user := types.GetUserIdentifier(ctx.Message.From)

	ctx.Deps.Logger.Info("Processing command",
		zap.String("request_id", requestCtx.RequestID),
		zap.String("command", requestCtx.Command),
		zap.Int64("chat_id", requestCtx.ChatID),
		zap.String("user", user),
		zap.Int("update_id", ctx.UpdateID))

	// Выполняем следующий middleware/handler
	err := next(ctx)

	// Логируем завершение обработки
	duration := time.Since(requestCtx.StartTime)
	if err != nil {
		ctx.Deps.Logger.Error("Command completed with error",
			zap.String("request_id", requestCtx.RequestID),
			zap.String("command", requestCtx.Command),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		ctx.Deps.Logger.Info("Command completed successfully",
			zap.String("request_id", requestCtx.RequestID),
			zap.String("command", requestCtx.Command),
			zap.Duration("duration", duration))
	}

	return err
}

// Debounce prevents double-clicks with context timeout
func Debounce(ctx types.Context, next types.HandlerFunc) error {
	key := fmt.Sprintf("%d:%s", ctx.Message.Chat.ID, ctx.Message.Command())

	if !ctx.Deps.Debouncer.CanProcessRequest(key) {
		user := types.GetUserIdentifier(ctx.Message.From)
		ctx.Deps.Logger.Info("Command debounced",
			zap.String("command", ctx.Message.Command()),
			zap.Int64("chat_id", ctx.Message.Chat.ID),
			zap.String("user", user),
			zap.Int("update_id", ctx.UpdateID))

		// Отправляем уведомление пользователю о дебаунсе
		if err := ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID,
			"⏱️ Пожалуйста, подождите перед повторным выполнением команды"); err != nil {
			ctx.Deps.Logger.Error("Failed to send debounce message", zap.Error(err))
		}

		return nil
	}

	return next(ctx)
}

// ErrorHandler handles errors from handlers with better context
func ErrorHandler(ctx types.Context, next types.HandlerFunc) error {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			user := types.GetUserIdentifier(ctx.Message.From)
			ctx.Deps.Logger.Error("Panic recovered in error handler",
				zap.String("command", ctx.Message.Command()),
				zap.Int64("chat_id", ctx.Message.Chat.ID),
				zap.String("user", user),
				zap.Int("update_id", ctx.UpdateID),
				zap.Any("panic", panicErr))

			// Отправляем пользователю сообщение об ошибке
			if err := ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID,
				"❌ Произошла серьезная ошибка. Попробуйте позже."); err != nil {
				ctx.Deps.Logger.Error("Failed to send panic message", zap.Error(err))
			}
		}
	}()

	err := next(ctx)
	if err != nil {
		user := types.GetUserIdentifier(ctx.Message.From)

		// Определяем тип ошибки для лучшего логирования
		var errorMessage string
		switch {
		case types.IsCommandError(err):
			ctx.Deps.Logger.Warn("Command error",
				zap.String("command", ctx.Message.Command()),
				zap.Int64("chat_id", ctx.Message.Chat.ID),
				zap.String("user", user),
				zap.Int("update_id", ctx.UpdateID),
				zap.Error(err))
			errorMessage = "❌ Ошибка выполнения команды"

		case types.IsBotError(err):
			ctx.Deps.Logger.Error("Bot error",
				zap.String("command", ctx.Message.Command()),
				zap.Int64("chat_id", ctx.Message.Chat.ID),
				zap.String("user", user),
				zap.Int("update_id", ctx.UpdateID),
				zap.Error(err))
			errorMessage = "🤖 Внутренняя ошибка бота"

		default:
			ctx.Deps.Logger.Error("Unknown error",
				zap.String("command", ctx.Message.Command()),
				zap.Int64("chat_id", ctx.Message.Chat.ID),
				zap.String("user", user),
				zap.Int("update_id", ctx.UpdateID),
				zap.Error(err))
			errorMessage = "❌ Неизвестная ошибка"
		}

		// Отправляем пользователю информативное сообщение об ошибке
		if sendErr := ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, errorMessage); sendErr != nil {
			ctx.Deps.Logger.Error("Failed to send error message", zap.Error(sendErr))
		}
	}

	return err
}

// AdminOnly restricts access to admin users with better validation
func AdminOnly(adminUsername string) func(ctx types.Context, next types.HandlerFunc) error {
	return func(ctx types.Context, next types.HandlerFunc) error {
		if ctx.Message.From == nil {
			ctx.Deps.Logger.Warn("No user information in message")
			return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID,
				"❌ Невозможно определить пользователя")
		}

		if ctx.Message.From.UserName != adminUsername {
			user := types.GetUserIdentifier(ctx.Message.From)
			ctx.Deps.Logger.Warn("Unauthorized access attempt",
				zap.String("command", ctx.Message.Command()),
				zap.String("user", user),
				zap.String("expected_admin", adminUsername))

			return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID,
				"🔒 Эта команда доступна только администратору")
		}

		return next(ctx)
	}
}

// Wrap wraps a middleware and handler into a HandlerFunc (backward compatibility)
func Wrap(mw func(ctx types.Context, next types.HandlerFunc) error, handler types.HandlerFunc) types.HandlerFunc {
	return func(ctx types.Context) error {
		return mw(ctx, handler)
	}
}

// MetricsMiddleware записывает метрики выполнения команд с улучшенной обработкой
func MetricsMiddleware(ctx types.Context, next types.HandlerFunc) error {
	if ctx.Deps.Metrics == nil {
		// Если метрики отключены, просто выполняем следующий handler
		return next(ctx)
	}

	startTime := time.Now()
	command := ctx.Message.Command()
	userID := ctx.Message.From.ID

	// Записываем команду пользователя
	ctx.Deps.Metrics.RecordUserCommand(command, userID)

	// Выполняем команду
	err := next(ctx)

	// Записываем время выполнения
	duration := time.Since(startTime)
	ctx.Deps.Metrics.RecordResponseTime(duration)

	// Записываем ошибку если есть
	if err != nil {
		ctx.Deps.Metrics.RecordError()
	}

	return err
}
