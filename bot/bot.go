package bot

import (
	"fmt"
	"os"

	"new_parser/parser"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Config holds the bot's configuration
type Config struct {
	AdminUsername string
	BotToken      string
}

// NewConfig creates a new Config instance by reading environment variables
func NewConfig() (*Config, error) {
	adminUsername := os.Getenv("ADMIN_USERNAME")
	if adminUsername == "" {
		adminUsername = "fullofsarang" // Значение по умолчанию
	}

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is not set in .env")
	}

	return &Config{
		AdminUsername: adminUsername,
		BotToken:      botToken,
	}, nil
}

// Bot represents the Telegram bot
type Bot struct {
	api      *tgbotapi.BotAPI
	logger   *zap.Logger
	handlers *CommandHandlers
	config   *Config
}

// NewBot creates a new bot instance
func NewBot(config *Config, logger *zap.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %v", err)
	}

	// Инициализируем Debouncer для защиты от дабл-клика
	debouncer := NewDebouncer()

	// Создаём CommandHandlers с необходимыми зависимостями
	handlers := NewCommandHandlers(api, logger, debouncer, config)

	return &Bot{
		api:      api,
		logger:   logger,
		handlers: handlers,
		config:   config,
	}, nil
}

// Start runs the bot
func (h *Bot) Start() error {
	defer h.handlers.keyboard.Stop() // Останавливаем тикер при завершении работы бота

	h.logger.Info("Bot started", zap.String("username", h.api.Self.UserName))

	// Инициализируем кэш асинхронно
	go parser.InitializeCache(h.logger)

	// Запускаем фоновое обновление кэша
	go StartCacheUpdater(h.logger)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := h.api.GetUpdates(tgbotapi.NewUpdate(0))
	if err != nil {
		return fmt.Errorf("failed to get updates: %v", err)
	}
	if len(updates) > 0 {
		u.Offset = updates[len(updates)-1].UpdateID + 1
	}

	updatesChan := h.api.GetUpdatesChan(u)

	for update := range updatesChan {
		// Handle Commands and Callback Queries
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		// Устанавливаем команды для всех пользователей
		if err := h.handlers.SetBotCommands(); err != nil {
			h.logger.Error("Failed to set bot commands", zap.Error(err))
		}

		// Handle Callback Queries (Inline Keyboard)
		if update.CallbackQuery != nil {
			go h.handlers.HandleCallbackQuery(update)
			continue
		}

		// Обработка команд
		if !update.Message.IsCommand() {
			continue
		}

		go h.handlers.HandleCommand(update)
	}

	return nil
}
