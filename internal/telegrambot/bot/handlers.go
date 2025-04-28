package bot

import (
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	"gemfactory/internal/telegrambot/bot/keyboard"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/pkg/config"
	"go.uber.org/zap"
)

// NewCommandHandlers creates a new CommandHandlers instance
func NewCommandHandlers(api botapi.BotAPI, logger *zap.Logger, debouncer *debounce.Debouncer, config *config.Config, al *artistlist.ArtistList, cache cache.Cache) *types.CommandHandlers {
	keyboard := keyboard.NewKeyboardManager(api, logger, al, config, cache)
	return &types.CommandHandlers{
		API:        api,
		Logger:     logger,
		Config:     config,
		ArtistList: al,
		Keyboard:   keyboard,
		Debouncer:  debouncer,
		Cache:      cache,
	}
}
