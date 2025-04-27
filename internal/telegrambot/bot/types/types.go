package types

import (
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/keyboard"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/pkg/config"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"strings"
)

// CommandHandlers handles Telegram commands
type CommandHandlers struct {
	API        *tgbotapi.BotAPI
	Logger     *zap.Logger
	Config     *config.Config
	ArtistList *artistlist.ArtistList
	Keyboard   *keyboard.KeyboardManager
	Debouncer  *debounce.Debouncer
}

// ParseArtists parses a comma-separated list of artists
func ParseArtists(input string) []string {
	rawArtists := strings.Split(input, ",")
	var artists []string
	for _, artist := range rawArtists {
		cleaned := strings.TrimSpace(artist)
		if cleaned != "" {
			artists = append(artists, cleaned)
		}
	}
	return artists
}

// SendMessage sends a simple text message
func SendMessage(h *CommandHandlers, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := h.API.Send(msg); err != nil {
		h.Logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.String("text", text), zap.Error(err))
	}
}

// SendMessageWithMarkup sends a message with a reply markup
func SendMessageWithMarkup(h *CommandHandlers, chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = markup
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true
	if _, err := h.API.Send(msg); err != nil {
		h.Logger.Error("Failed to send message with markup", zap.Int64("chat_id", chatID), zap.String("text", text), zap.Error(err))
	}
}

// SetBotCommands sets the bot's command menu
func (h *CommandHandlers) SetBotCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "/help", Description: "Показать справку"},
		{Command: "/month", Description: "Получить релизы за месяц"},
		{Command: "/whitelists", Description: "Показать списки артистов"},
	}

	config := tgbotapi.NewSetMyCommands(commands...)
	if _, err := h.API.Request(config); err != nil {
		return err
	}
	h.Logger.Info("Bot commands set successfully")
	return nil
}
