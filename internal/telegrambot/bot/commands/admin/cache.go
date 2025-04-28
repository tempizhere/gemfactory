package admin

import (
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// HandleClearCache processes the /clearcache command
func HandleClearCache(h *types.CommandHandlers, msg *tgbotapi.Message) {
	svc := service.NewReleaseService(h.ArtistList, h.Config, h.Logger)
	svc.ClearCache()
	if err := h.API.SendMessage(msg.Chat.ID, "Кэш очищен, обновление запущено."); err != nil {
		h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Кэш очищен, обновление запущено."), zap.Error(err))
	}
}

// HandleClearWhitelists processes the /clearwhitelists command
func HandleClearWhitelists(h *types.CommandHandlers, msg *tgbotapi.Message) {
	svc := service.NewArtistService(h.ArtistList, h.Logger)
	if err := svc.ClearWhitelists(); err != nil {
		if err := h.API.SendMessage(msg.Chat.ID, err.Error()); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", err.Error()), zap.Error(err))
		}
		return
	}
	if err := h.API.SendMessage(msg.Chat.ID, "Вайтлисты очищены"); err != nil {
		h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Вайтлисты очищены"), zap.Error(err))
	}
}
