package types

import (
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	"gemfactory/internal/telegrambot/bot/keyboard"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/pkg/config"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"strings"
)

// CommandHandlers handles Telegram commands
type CommandHandlers struct {
	API        botapi.BotAPI // Используем интерфейс BotAPI из пакета botapi
	Logger     *zap.Logger
	Config     *config.Config
	ArtistList *artistlist.ArtistList
	Keyboard   *keyboard.KeyboardManager
	Debouncer  *debounce.Debouncer
	Cache      cache.Cache
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

// SetBotCommands sets the bot's command menu
func (h *CommandHandlers) SetBotCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "/help", Description: "Показать справку"},
		{Command: "/month", Description: "Получить релизы за месяц"},
		{Command: "/whitelists", Description: "Показать списки артистов"},
	}

	if err := h.API.SetBotCommands(commands); err != nil {
		return err
	}
	h.Logger.Info("Bot commands set successfully")
	return nil
}
