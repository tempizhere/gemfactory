package bot

import (
	"fmt"
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	"gemfactory/internal/telegrambot/bot/commands"
	"gemfactory/internal/telegrambot/bot/keyboard"
	"gemfactory/internal/telegrambot/bot/middleware"
	"gemfactory/internal/telegrambot/bot/router"
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/releases/artist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/scraper"
	"gemfactory/internal/telegrambot/releases/updater"
	"gemfactory/pkg/config"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Bot represents the Telegram bot
type Bot struct {
	api      botapi.BotAPI
	logger   *zap.Logger
	config   *config.Config
	router   *router.Router
	deps     *types.Dependencies
	keyboard *keyboard.KeyboardManager
}

// NewBot creates a new bot instance
func NewBot(config *config.Config, logger *zap.Logger) (*Bot, error) {
	tgApi, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}

	api := botapi.NewTelegramBotAPI(tgApi)
	al := artist.NewWhitelistManager(config.WhitelistDir, logger)

	if len(al.GetFemaleWhitelist()) == 0 && len(al.GetMaleWhitelist()) == 0 {
		logger.Warn("Both female and male whitelists are empty; populate at least one whitelist using /add_artist")
	}

	scraper := scraper.NewFetcher(config, logger)
	cacheManager := cache.NewCacheManager(config, logger, al, scraper, nil)
	updater := updater.NewUpdater(config, logger, al, cacheManager, scraper)
	cacheManager.SetUpdater(updater)

	debouncer := debounce.NewDebouncer()
	releaseService := service.NewReleaseService(al, config, logger, cacheManager)
	artistService := service.NewArtistService(al, logger)
	keyboardManager := keyboard.NewKeyboardManager(api, logger, al, config, cacheManager)

	deps := &types.Dependencies{
		BotAPI:         api,
		Logger:         logger,
		Config:         config,
		ReleaseService: releaseService,
		ArtistService:  artistService,
		Keyboard:       keyboardManager,
		Debouncer:      debouncer,
		Cache:          cacheManager,
	}

	r := router.NewRouter()
	r.Use(middleware.LogRequest)
	r.Use(middleware.Debounce)
	r.Use(middleware.ErrorHandler)

	logger.Info("Initializing command routes")
	commands.RegisterRoutes(r, deps)

	bot := &Bot{
		api:      api,
		logger:   logger,
		config:   config,
		router:   r,
		deps:     deps,
		keyboard: keyboardManager,
	}

	if len(al.GetFemaleWhitelist()) > 0 || len(al.GetMaleWhitelist()) > 0 {
		logger.Info("Starting cache updater")
		go cacheManager.StartUpdater()
	} else {
		logger.Warn("Cache updater not started due to empty whitelists")
	}

	return bot, nil
}

// Start runs the bot
func (b *Bot) Start() error {
	defer b.keyboard.Stop()

	tgApi := b.api.(*botapi.TelegramBotAPI).GetAPI()
	b.logger.Info("Bot started", zap.String("username", tgApi.Self.UserName))

	_, err := tgApi.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	if err != nil {
		b.logger.Error("Failed to delete webhook", zap.Error(err))
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	if err := b.deps.SetBotCommands(); err != nil {
		b.logger.Error("Failed to set bot commands", zap.Error(err))
		return fmt.Errorf("failed to set bot commands: %w", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}

	b.logger.Info("Starting to fetch updates")
	updatesChan := tgApi.GetUpdatesChan(u)
	if updatesChan == nil {
		return fmt.Errorf("failed to create updates channel")
	}

	for update := range updatesChan {
		if update.Message != nil {
			b.logger.Debug("Received message",
				zap.String("text", update.Message.Text),
				zap.Int64("chat_id", update.Message.Chat.ID),
				zap.String("user", types.GetUserIdentifier(update.Message.From)),
				zap.Int("update_id", update.UpdateID))
		} else if update.CallbackQuery != nil {
			month := extractMonth(update.CallbackQuery.Data)
			b.logger.Info("Received callback",
				zap.String("data", update.CallbackQuery.Data),
				zap.String("month", month),
				zap.Int64("chat_id", update.CallbackQuery.Message.Chat.ID),
				zap.String("user", types.GetUserIdentifier(update.CallbackQuery.From)))
			b.logger.Debug("Callback details",
				zap.Int("update_id", update.UpdateID))
		}

		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		if update.CallbackQuery != nil {
			go b.keyboard.HandleCallbackQuery(update.CallbackQuery)
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		go b.handleUpdate(update)
	}

	b.logger.Info("Update channel closed")
	return nil
}

// handleUpdate processes incoming updates
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	ctx := types.Context{
		Message:  update.Message,
		UpdateID: update.UpdateID,
		Deps:     b.deps,
	}
	if err := b.router.Dispatch(ctx); err != nil {
		b.logger.Error("Failed to dispatch command",
			zap.String("command", ctx.Message.Command()),
			zap.Int64("chat_id", ctx.Message.Chat.ID),
			zap.String("user", types.GetUserIdentifier(ctx.Message.From)),
			zap.Int("update_id", ctx.UpdateID),
			zap.Error(err))
	}
}

// extractMonth extracts the month from callback data
func extractMonth(data string) string {
	if strings.HasPrefix(data, "month_") {
		return strings.TrimPrefix(data, "month_")
	}
	return "unknown"
}