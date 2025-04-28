package bot

import (
	"fmt"
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	"gemfactory/internal/telegrambot/bot/commands/admin"
	"gemfactory/internal/telegrambot/bot/commands/user"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/pkg/config"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"strings"
)

// Bot represents the Telegram bot
type Bot struct {
	api      botapi.BotAPI
	logger   *zap.Logger
	handlers *types.CommandHandlers
	config   *config.Config
	al       *artistlist.ArtistList
}

// NewConfig creates a new Config instance by reading environment variables
func NewConfig() (*config.Config, error) {
	return config.Load()
}

// NewBot creates a new bot instance
func NewBot(config *config.Config, logger *zap.Logger) (*Bot, error) {
	tgApi, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %v", err)
	}

	// Оборачиваем tgbotapi.BotAPI в TelegramBotAPI
	api := NewTelegramBotAPI(tgApi)

	// Инициализируем ArtistList
	al, err := artistlist.NewArtistList(config.WhitelistDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize artist list: %v", err)
	}

	// Проверяем, пусты ли вайтлисты
	if len(al.GetFemaleWhitelist()) == 0 && len(al.GetMaleWhitelist()) == 0 {
		return nil, fmt.Errorf("both female and male whitelists are empty; populate at least one whitelist to start the bot")
	}

	// Инициализируем Debouncer для защиты от дабл-клика
	debouncer := debounce.NewDebouncer()

	// Создаём CommandHandlers с необходимыми зависимостями
	handlers := NewCommandHandlers(api, logger, debouncer, config, al)

	bot := &Bot{
		api:      api,
		logger:   logger,
		handlers: handlers,
		config:   config,
		al:       al,
	}

	// Инициализируем конфигурацию кэша
	bot.logger.Info("Initializing cache configuration")
	cache.InitCacheConfig(bot.logger)

	// Запускаем периодическое обновление кэша
	bot.logger.Info("Starting cache updater")
	go cache.StartUpdater(bot.config, bot.logger, bot.al)

	return bot, nil
}

// Start runs the bot
func (b *Bot) Start() error {
	defer b.handlers.Keyboard.Stop() // Останавливаем тикер при завершении работы бота

	// Получаем имя бота через Telegram API
	tgApi := b.api.(*TelegramBotAPI).api
	b.logger.Info("Bot started", zap.String("username", tgApi.Self.UserName))

	// Отключаем вебхук и очищаем очередь обновлений
	_, err := tgApi.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
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

	// Настраиваем обновления
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}

	b.logger.Info("Starting to fetch updates from Telegram API")
	updatesChan := tgApi.GetUpdatesChan(u)
	if err != nil {
		b.logger.Error("Failed to create updates channel")
		return fmt.Errorf("failed to create updates channel")
	}

	b.logger.Info("Listening for updates")
	for update := range updatesChan {
		if update.Message != nil {
			b.logger.Info("Received command", zap.String("text", update.Message.Text))
		} else if update.CallbackQuery != nil {
			b.logger.Info("Received callback", zap.String("data", update.CallbackQuery.Data))
		}

		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		if update.CallbackQuery != nil {
			go b.handlers.Keyboard.HandleCallbackQuery(update.CallbackQuery)
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		go b.handleCommand(update)
	}

	b.logger.Info("Update channel closed, stopping bot")
	return nil
}

// handleCommand processes incoming commands
func (b *Bot) handleCommand(update tgbotapi.Update) {
	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	msg := update.Message
	command := msg.Command()
	args := strings.Fields(msg.Text)[1:]

	isAdmin := msg.From.UserName == b.config.AdminUsername

	switch command {
	case "start":
		user.HandleStart(b.handlers, msg)
	case "help":
		user.HandleHelp(b.handlers, msg)
	case "month":
		user.HandleMonth(b.handlers, msg, args)
	case "whitelists":
		if isAdmin {
			admin.HandleWhitelists(b.handlers, msg)
		} else {
			b.handlers.API.SendMessage(msg.Chat.ID, "Эта команда доступна только администратору.")
		}
	case "add_artist":
		if isAdmin {
			admin.HandleAddArtist(b.handlers, msg, args)
		} else {
			b.handlers.API.SendMessage(msg.Chat.ID, "Эта команда доступна только администратору.")
		}
	case "remove_artist":
		if isAdmin {
			admin.HandleRemoveArtist(b.handlers, msg, args)
		} else {
			b.handlers.API.SendMessage(msg.Chat.ID, "Эта команда доступна только администратору.")
		}
	case "clearcache":
		if isAdmin {
			admin.HandleClearCache(b.handlers, msg)
		} else {
			b.handlers.API.SendMessage(msg.Chat.ID, "Эта команда доступна только администратору.")
		}
	case "clearwhitelists":
		if isAdmin {
			admin.HandleClearWhitelists(b.handlers, msg)
		} else {
			b.handlers.API.SendMessage(msg.Chat.ID, "Эта команда доступна только администратору.")
		}
	case "export":
		if isAdmin {
			admin.HandleExport(b.handlers, msg)
		} else {
			b.handlers.API.SendMessage(msg.Chat.ID, "Эта команда доступна только администратору.")
		}
	default:
		b.handlers.API.SendMessage(msg.Chat.ID, "Неизвестная команда. Используйте /help для списка команд.")
	}
}
