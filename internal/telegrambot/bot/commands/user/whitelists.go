package user

import (
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// HandleWhitelists processes the /whitelists command
func HandleWhitelists(h *types.CommandHandlers, msg *tgbotapi.Message) {
	svc := service.NewArtistService(h.ArtistList, h.Logger)
	response := svc.FormatWhitelists()
	if err := h.API.SendMessageWithMarkup(msg.Chat.ID, response, h.Keyboard.GetMainKeyboard()); err != nil {
		h.Logger.Error("Failed to send message with markup", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", response), zap.Error(err))
	}
}
