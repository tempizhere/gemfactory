package botapi

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// BotAPI defines the interface for interacting with Telegram API
type BotAPI interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithMarkup(chatID int64, text string, markup any) error
	EditMessageReplyMarkup(chatID int64, messageID int, markup any) error
	SetBotCommands(commands []tgbotapi.BotCommand) error
}

// TelegramBotAPI wraps tgbotapi.BotAPI to implement the BotAPI interface
// Теперь logger обязателен
type TelegramBotAPI struct {
	api    *tgbotapi.BotAPI
	logger *zap.Logger
}

// NewTelegramBotAPI creates a new TelegramBotAPI instance
func NewTelegramBotAPI(api *tgbotapi.BotAPI, logger *zap.Logger) *TelegramBotAPI {
	return &TelegramBotAPI{
		api:    api,
		logger: logger,
	}
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
		t.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.Error(err))
	}
	return err
}

// SendMessageWithMarkup sends a message with a reply markup
func (t *TelegramBotAPI) SendMessageWithMarkup(chatID int64, text string, markup any) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = markup
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true
	_, err := t.api.Send(msg)
	if err != nil {
		t.logger.Error("Failed to send message with markup", zap.Int64("chat_id", chatID), zap.Error(err))
	}
	return err
}

// EditMessageReplyMarkup edits the reply markup of a message
func (t *TelegramBotAPI) EditMessageReplyMarkup(chatID int64, messageID int, markup any) error {
	inlineMarkup, ok := markup.(tgbotapi.InlineKeyboardMarkup)
	if !ok {
		return fmt.Errorf("markup must be of type tgbotapi.InlineKeyboardMarkup")
	}
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, inlineMarkup)
	_, err := t.api.Send(edit)
	if err != nil {
		t.logger.Error("Failed to edit message reply markup", zap.Int64("chat_id", chatID), zap.Int("message_id", messageID), zap.Error(err))
	}
	return err
}

// SetBotCommands sets the bot's command menu
func (t *TelegramBotAPI) SetBotCommands(commands []tgbotapi.BotCommand) error {
	_, err := t.api.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		t.logger.Error("Failed to set bot commands", zap.Error(err))
	}
	return err
}

var _ BotAPI = (*TelegramBotAPI)(nil)
