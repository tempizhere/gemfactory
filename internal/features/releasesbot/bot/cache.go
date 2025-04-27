package bot

import (
	"fmt"
	"gemfactory/internal/features/releasesbot/cache"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleClearCache processes the /clearcache command
func handleClearCache(h *CommandHandlers, msg *tgbotapi.Message) {
	cache.ClearCache()
	go cache.InitializeCache(h.config, h.logger, h.al)
	sendMessage(h, msg.Chat.ID, "Кэш очищен, обновление запущено.")
}

// handleClearWhitelists processes the /clearwhitelists command
func handleClearWhitelists(h *CommandHandlers, msg *tgbotapi.Message) {
	if err := h.al.ClearWhitelists(); err != nil {
		sendMessage(h, msg.Chat.ID, fmt.Sprintf("Ошибка при очистке вайтлистов: %v", err))
		return
	}
	sendMessage(h, msg.Chat.ID, "Вайтлисты очищены")
}
