package admin

import (
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleExport processes the /export command
func HandleExport(h *types.CommandHandlers, msg *tgbotapi.Message) {
	svc := service.NewArtistService(h.ArtistList, h.Logger)
	response := svc.FormatWhitelistsForExport()
	h.API.SendMessageWithMarkup(msg.Chat.ID, response, h.Keyboard.GetMainKeyboard())
}
