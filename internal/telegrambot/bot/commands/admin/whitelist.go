package admin

import (
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleWhitelists processes the /whitelists command
func HandleWhitelists(h *types.CommandHandlers, msg *tgbotapi.Message) {
	svc := service.NewArtistService(h.ArtistList, h.Logger)
	response := svc.FormatWhitelists()
	h.API.SendMessageWithMarkup(msg.Chat.ID, response, h.Keyboard.GetMainKeyboard())
}
