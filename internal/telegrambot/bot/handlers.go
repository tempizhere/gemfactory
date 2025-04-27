package bot

import (
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/keyboard"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/pkg/config"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// NewCommandHandlers creates a new CommandHandlers instance
func NewCommandHandlers(api *tgbotapi.BotAPI, logger *zap.Logger, debouncer *debounce.Debouncer, config *config.Config, al *artistlist.ArtistList) *types.CommandHandlers {
	keyboard := keyboard.NewKeyboardManager(api, logger, al, config)
	return &types.CommandHandlers{
		API:        api,
		Logger:     logger,
		Config:     config,
		ArtistList: al,
		Keyboard:   keyboard,
		Debouncer:  debouncer,
	}
}
