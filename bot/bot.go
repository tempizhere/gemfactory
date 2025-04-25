package bot

import (
	"fmt"
	"os"

	"gemfactory/parser"

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
func (b *Bot) Start() error {
	defer b.handlers.keyboard.Stop() // Останавливаем тикер при завершении работы бота

	b.logger.Info("Bot started", zap.String("username", b.api.Self.UserName))

	// Отключаем вебхук и очищаем очередь обновлений
	_, err := b.api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	if err != nil {
		b.logger.Error("Failed to delete webhook", zap.Error(err))
		return fmt.Errorf("failed to delete webhook: %v", err)
	}
	b.logger.Info("Webhook removed successfully")

	// Устанавливаем команды один раз при запуске
	if err := b.handlers.SetBotCommands(); err != nil {
		b.logger.Error("Failed to set bot commands", zap.Error(err))
		return fmt.Errorf("failed to set bot commands: %v", err)
	}

	// Инициализируем кэш асинхронно
	go parser.InitializeCache(b.logger)

	// Настраиваем обновления
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"} // Явно указываем типы обновлений

	b.logger.Info("Starting to fetch updates from Telegram API")
	updatesChan := b.api.GetUpdatesChan(u)
	if updatesChan == nil {
		b.logger.Error("Failed to create updates channel")
		return fmt.Errorf("failed to create updates channel")
	}

	b.logger.Info("Listening for updates")
	for update := range updatesChan {
		// Логируем получение обновления с минимальными деталями
		if update.Message != nil {
			b.logger.Info("Received command", zap.String("text", update.Message.Text))
		} else if update.CallbackQuery != nil {
			b.logger.Info("Received callback", zap.String("data", update.CallbackQuery.Data))
		}

		// Handle Commands and Callback Queries
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		// Handle Callback Queries (Inline Keyboard)
		if update.CallbackQuery != nil {
			go b.handlers.HandleCallbackQuery(update)
			continue
		}

		// Обработка команд
		if !update.Message.IsCommand() {
			continue
		}

		go b.handlers.HandleCommand(update)
	}

	b.logger.Info("Update channel closed, stopping bot")
	return nil
}
