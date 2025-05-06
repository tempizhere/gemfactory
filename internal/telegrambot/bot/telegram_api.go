package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gemfactory/internal/telegrambot/bot/botapi"
)

// TelegramBotAPI wraps tgbotapi.BotAPI to implement the BotAPI interface
type TelegramBotAPI struct {
	api *tgbotapi.BotAPI
}

// NewTelegramBotAPI creates a new TelegramBotAPI instance
func NewTelegramBotAPI(api *tgbotapi.BotAPI) *TelegramBotAPI {
	return &TelegramBotAPI{api: api}
}

// SendMessage sends a simple text message
func (t *TelegramBotAPI) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := t.api.Send(msg)
	return err
}

// SendMessageWithMarkup sends a message with a reply markup
func (t *TelegramBotAPI) SendMessageWithMarkup(chatID int64, text string, markup interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = markup
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true
	_, err := t.api.Send(msg)
	return err
}

// EditMessageReplyMarkup edits the reply markup of a message
func (t *TelegramBotAPI) EditMessageReplyMarkup(chatID int64, messageID int, markup interface{}) error {
	inlineMarkup, ok := markup.(tgbotapi.InlineKeyboardMarkup)
	if !ok {
		return fmt.Errorf("markup must be of type tgbotapi.InlineKeyboardMarkup")
	}
	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, inlineMarkup)
	_, err := t.api.Request(msg)
	return err
}

// SetBotCommands sets the bot's command menu
func (t *TelegramBotAPI) SetBotCommands(commands []tgbotapi.BotCommand) error {
	config := tgbotapi.NewSetMyCommands(commands...)
	_, err := t.api.Request(config)
	return err
}

var _ botapi.BotAPI = (*TelegramBotAPI)(nil) // Проверяем, что TelegramBotAPI реализует BotAPI
