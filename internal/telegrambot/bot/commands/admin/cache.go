package admin

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/releases/cache"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleClearCache processes the /clearcache command
func HandleClearCache(h *types.CommandHandlers, msg *tgbotapi.Message) {
	cache.ClearCache()
	go cache.InitializeCache(h.Config, h.Logger, h.ArtistList)
	types.SendMessage(h, msg.Chat.ID, "Кэш очищен, обновление запущено.")
}

// HandleClearWhitelists processes the /clearwhitelists command
func HandleClearWhitelists(h *types.CommandHandlers, msg *tgbotapi.Message) {
	if err := h.ArtistList.ClearWhitelists(); err != nil {
		types.SendMessage(h, msg.Chat.ID, fmt.Sprintf("Ошибка при очистке вайтлистов: %v", err))
		return
	}
	types.SendMessage(h, msg.Chat.ID, "Вайтлисты очищены")
}
