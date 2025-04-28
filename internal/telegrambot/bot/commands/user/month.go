package user

import (
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

// HandleMonth processes the /month command
func HandleMonth(h *types.CommandHandlers, msg *tgbotapi.Message, args []string) {
	if len(args) == 0 {
		h.API.SendMessageWithMarkup(msg.Chat.ID, "Пожалуйста, выберите месяц:", h.Keyboard.GetMainKeyboard())
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

	svc := service.NewReleaseService(h.ArtistList, h.Config, h.Logger)
	response, err := svc.GetReleasesForMonth(month, femaleOnly, maleOnly)
	if err != nil {
		h.API.SendMessage(msg.Chat.ID, err.Error())
		return
	}

	h.API.SendMessageWithMarkup(msg.Chat.ID, response, h.Keyboard.GetMainKeyboard())
}
