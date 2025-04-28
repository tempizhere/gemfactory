package botapi

import "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// BotAPI defines the interface for interacting with Telegram API
type BotAPI interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithMarkup(chatID int64, text string, markup interface{}) error
	EditMessageReplyMarkup(chatID int64, messageID int, markup interface{}) error
	SetBotCommands(commands []tgbotapi.BotCommand) error
}
