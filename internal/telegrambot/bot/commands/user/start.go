package user

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleStart processes the /start command
func HandleStart(h *types.CommandHandlers, msg *tgbotapi.Message) {
	text := "Добро пожаловать! Выберите месяц:"
	h.API.SendMessageWithMarkup(msg.Chat.ID, text, h.Keyboard.GetMainKeyboard())
}

// HandleHelp processes the /help command
func HandleHelp(h *types.CommandHandlers, msg *tgbotapi.Message) {
	text := "Доступные команды:\n" +
		"\n/start - Начать работу с ботом\n" +
		"/help - Показать это сообщение\n" +
		"/month [месяц] - Получить релизы за указанный месяц\n" +
		"/month [месяц] -gg - Получить релизы только для женских групп\n" +
		"/month [месяц] -mg - Получить релизы только для мужских групп\n" +
		"/whitelists - Показать списки артистов\n" +
		"\n" +
		fmt.Sprintf("По вопросам вайтлистов обращайтесь к @%s", h.Config.AdminUsername)
	h.API.SendMessageWithMarkup(msg.Chat.ID, text, h.Keyboard.GetMainKeyboard())
}
