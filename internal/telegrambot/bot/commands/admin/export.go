package admin

import (
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// HandleExport processes the /export command
func HandleExport(h *types.CommandHandlers, msg *tgbotapi.Message) {
	svc := service.NewArtistService(h.ArtistList, h.Logger)
	response := svc.FormatWhitelistsForExport()
	if err := h.API.SendMessageWithMarkup(msg.Chat.ID, response, h.Keyboard.GetMainKeyboard()); err != nil {
		h.Logger.Error("Failed to send message with markup", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", response), zap.Error(err))
	}
}
