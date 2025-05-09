package botapi

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BotAPI defines the interface for interacting with Telegram API
type BotAPI interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithMarkup(chatID int64, text string, markup interface{}) error
	EditMessageReplyMarkup(chatID int64, messageID int, markup interface{}) error
	SetBotCommands(commands []tgbotapi.BotCommand) error
}

// TelegramBotAPI wraps tgbotapi.BotAPI to implement the BotAPI interface
type TelegramBotAPI struct {
	api *tgbotapi.BotAPI
}

// NewTelegramBotAPI creates a new TelegramBotAPI instance
func NewTelegramBotAPI(api *tgbotapi.BotAPI) *TelegramBotAPI {
	return &TelegramBotAPI{api: api}
}

// GetAPI returns the underlying tgbotapi.BotAPI instance
func (t *TelegramBotAPI) GetAPI() *tgbotapi.BotAPI {
	return t.api
}

// SendMessage sends a simple text message
func (t *TelegramBotAPI) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := t.api.Send(msg)
	if err != nil {
		fmt.Printf("Failed to send message to chat %d: %v\n", chatID, err)
	}
	return err
}

// SendMessageWithMarkup sends a message with a reply markup
func (t *TelegramBotAPI) SendMessageWithMarkup(chatID int64, text string, markup interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = markup
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true
	_, err := t.api.Send(msg)
	if err != nil {
		fmt.Printf("Failed to send message with markup to chat %d: %v\n", chatID, err)
	}
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
	if err != nil {
		fmt.Printf("Failed to edit message reply markup for chat %d: %v\n", chatID, err)
	}
	return err
}

// SetBotCommands sets the bot's command menu
func (t *TelegramBotAPI) SetBotCommands(commands []tgbotapi.BotCommand) error {
	config := tgbotapi.NewSetMyCommands(commands...)
	_, err := t.api.Request(config)
	if err != nil {
		fmt.Printf("Failed to set bot commands: %v\n", err)
	}
	return err
}

var _ BotAPI = (*TelegramBotAPI)(nil)
