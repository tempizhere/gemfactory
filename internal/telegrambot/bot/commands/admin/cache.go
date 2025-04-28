package admin

import (
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleClearCache processes the /clearcache command
func HandleClearCache(h *types.CommandHandlers, msg *tgbotapi.Message) {
	svc := service.NewReleaseService(h.ArtistList, h.Config, h.Logger)
	svc.ClearCache()
	h.API.SendMessage(msg.Chat.ID, "Кэш очищен, обновление запущено.")
}

// HandleClearWhitelists processes the /clearwhitelists command
func HandleClearWhitelists(h *types.CommandHandlers, msg *tgbotapi.Message) {
	svc := service.NewArtistService(h.ArtistList, h.Logger)
	if err := svc.ClearWhitelists(); err != nil {
		h.API.SendMessage(msg.Chat.ID, err.Error())
		return
	}
	h.API.SendMessage(msg.Chat.ID, "Вайтлисты очищены")
}
