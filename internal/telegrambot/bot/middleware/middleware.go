package middleware

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/types"

	"go.uber.org/zap"
)

// LogRequest logs incoming commands
func LogRequest(ctx types.Context, next types.HandlerFunc) error {
	user := types.GetUserIdentifier(ctx.Message.From)
	ctx.Deps.Logger.Info("Processing command",
		zap.String("command", ctx.Message.Command()),
		zap.Int64("chat_id", ctx.Message.Chat.ID),
		zap.String("user", user))
	ctx.Deps.Logger.Debug("Processing command details",
		zap.Int("update_id", ctx.UpdateID))
	return next(ctx)
}

// Debounce prevents double-clicks
func Debounce(ctx types.Context, next types.HandlerFunc) error {
	key := fmt.Sprintf("%d:%s:%d", ctx.Message.Chat.ID, ctx.Message.Command(), ctx.UpdateID)
	if !ctx.Deps.Debouncer.CanProcessRequest(key) {
		user := types.GetUserIdentifier(ctx.Message.From)
		ctx.Deps.Logger.Info("Command debounced",
			zap.String("command", ctx.Message.Command()),
			zap.Int64("chat_id", ctx.Message.Chat.ID),
			zap.String("user", user))
		ctx.Deps.Logger.Debug("Debounced command details",
			zap.Int("update_id", ctx.UpdateID))
		return nil
	}
	return next(ctx)
}

// ErrorHandler handles errors from handlers
func ErrorHandler(ctx types.Context, next types.HandlerFunc) error {
	err := next(ctx)
	if err != nil {
		user := types.GetUserIdentifier(ctx.Message.From)
		ctx.Deps.Logger.Error("Handler error",
			zap.String("command", ctx.Message.Command()),
			zap.Int64("chat_id", ctx.Message.Chat.ID),
			zap.String("user", user),
			zap.Int("update_id", ctx.UpdateID),
			zap.Error(err))
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, fmt.Sprintf("Ошибка: %v", err))
	}
	return nil
}

// AdminOnly restricts access to admin users
func AdminOnly(adminUsername string) func(ctx types.Context, next types.HandlerFunc) error {
	return func(ctx types.Context, next types.HandlerFunc) error {
		if ctx.Message.From.UserName != adminUsername {
			return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Эта команда доступна только администратору")
		}
		return next(ctx)
	}
}

// Wrap wraps a middleware and handler into a HandlerFunc
func Wrap(mw func(ctx types.Context, next types.HandlerFunc) error, handler types.HandlerFunc) types.HandlerFunc {
	return func(ctx types.Context) error {
		return mw(ctx, handler)
	}
}
