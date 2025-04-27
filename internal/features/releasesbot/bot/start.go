package bot

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleStart processes the /start command
func handleStart(h *CommandHandlers, msg *tgbotapi.Message) {
	text := "Добро пожаловать! Выберите месяц:"
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
	sendMessageWithMarkup(h, msg.Chat.ID, text, reply.ReplyMarkup)
}

// handleHelp processes the /help command
func handleHelp(h *CommandHandlers, msg *tgbotapi.Message) {
	text := "Доступные команды:\n" +
		"\n/start - Начать работу с ботом\n" +
		"/help - Показать это сообщение\n" +
		"/month [месяц] - Получить релизы за указанный месяц\n" +
		"/month [месяц] -gg - Получить релизы только для женских групп\n" +
		"/month [месяц] -mg - Получить релизы только для мужских групп\n" +
		"/whitelists - Показать списки артистов\n" +
		"\n" +
		fmt.Sprintf("По вопросам вайтлистов обращайтесь к @%s", h.config.AdminUsername)
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
	sendMessageWithMarkup(h, msg.Chat.ID, text, reply.ReplyMarkup)
}
