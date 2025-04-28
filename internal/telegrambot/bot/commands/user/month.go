package user

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"strings"
)

// HandleMonth processes the /month command
func HandleMonth(h *types.CommandHandlers, msg *tgbotapi.Message, args []string) {
	if len(args) == 0 {
		if err := h.API.SendMessageWithMarkup(msg.Chat.ID, "Пожалуйста, выберите месяц:", h.Keyboard.GetMainKeyboard()); err != nil {
			h.Logger.Error("Failed to send message with markup", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Пожалуйста, выберите месяц:"), zap.Error(err))
		}
		return
	}

	month := strings.ToLower(args[0])
	femaleOnly := false
	maleOnly := false

	for _, arg := range args[1:] {
		if arg == "-gg" {
			femaleOnly = true
		} else if arg == "-mg" {
			maleOnly = true
		}
	}

	svc := service.NewReleaseService(h.ArtistList, h.Config, h.Logger, h.Cache)
	response, err := svc.GetReleasesForMonth(month, femaleOnly, maleOnly)
	if err != nil {
		if err := h.API.SendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка: %v", err)); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", fmt.Sprintf("Ошибка: %v", err)), zap.Error(err))
		}
		return
	}

	if err := h.API.SendMessageWithMarkup(msg.Chat.ID, response, h.Keyboard.GetMainKeyboard()); err != nil {
		h.Logger.Error("Failed to send message with markup", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", response), zap.Error(err))
	}
}
