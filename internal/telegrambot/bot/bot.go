// Package bot содержит основную логику Telegram-бота.
package bot

import (
	"context"
	"fmt"
	"gemfactory/internal/telegrambot/bot/botapi"
	commandcache "gemfactory/internal/telegrambot/bot/cache"
	"gemfactory/internal/telegrambot/bot/commands"
	"gemfactory/internal/telegrambot/bot/health"
	"gemfactory/internal/telegrambot/bot/keyboard"
	"gemfactory/internal/telegrambot/bot/middleware"
	"gemfactory/internal/telegrambot/bot/router"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/bot/worker"
	"gemfactory/pkg/config"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Bot represents the main bot instance
type Bot struct {
	api          botapi.BotAPI
	logger       *zap.Logger
	config       config.Interface
	router       router.Interface
	deps         *types.Dependencies
	keyboard     keyboard.ManagerInterface
	workerPool   worker.PoolInterface
	health       health.ServerInterface
	commandCache commandcache.CommandCacheInterface
	rateLimiter  middleware.RateLimiterInterface
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// Убеждаемся, что Bot реализует BotInterface
var _ types.Interface = (*Bot)(nil)

// NewBot creates a new Bot instance
func NewBot(config *config.Config, logger *zap.Logger) (*Bot, error) {
	factory := NewComponentFactory(config, logger)

	// Создаем компоненты через фабрику
	api, err := factory.CreateBotAPI()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}

	whitelistManager := factory.CreateWhitelistManager()
	if len(whitelistManager.GetFemaleWhitelist()) == 0 && len(whitelistManager.GetMaleWhitelist()) == 0 {
		logger.Warn("Both female and male whitelists are empty; populate at least one whitelist using /add_artist")
	}

	scraper := factory.CreateScraper()
	workerPool := factory.CreateWorkerPool()
	commandCache := factory.CreateCommandCache()
	rateLimiter := factory.CreateRateLimiter()
	healthServer := factory.CreateServer()

	// Создаем зависимости
	deps := factory.CreateDependencies(
		api, whitelistManager, nil, nil, nil,
		nil, workerPool, commandCache,
	)

	// Создаем кэш с метриками
	cache := factory.CreateManager(whitelistManager, scraper, deps.Metrics)
	releaseService, artistService := factory.CreateServices(whitelistManager, cache)
	keyboardManager := factory.CreateKeyboardManager(api, whitelistManager, cache)

	// Обновляем зависимости с правильными сервисами
	deps.ReleaseService = releaseService
	deps.ArtistService = artistService
	deps.Cache = cache
	deps.Keyboard = keyboardManager

	// Настраиваем роутер
	r := router.NewRouter()
	r.Use(middleware.LogRequest)
	r.Use(middleware.MetricsMiddleware)
	r.Use(middleware.Debounce)
	r.Use(middleware.ErrorHandler)

	// Добавляем rate limiting middleware если включен
	if rateLimiter != nil {
		r.Use(createRateLimitMiddleware(rateLimiter, logger))
	}

	logger.Info("Initializing command routes")
	commands.RegisterRoutes(r, deps)

	bot := &Bot{
		api:          api,
		logger:       logger,
		config:       config,
		router:       r,
		deps:         deps,
		keyboard:     keyboardManager,
		workerPool:   workerPool,
		health:       healthServer,
		commandCache: commandCache,
		rateLimiter:  rateLimiter,
		stopChan:     make(chan struct{}),
	}

	// Устанавливаем время следующего обновления кэша
	if deps.Metrics != nil {
		// Получаем CACHE_DURATION из конфигурации
		cacheDuration := config.CacheDuration
		if cacheDuration <= 0 {
			cacheDuration = 8 * time.Hour // значение по умолчанию
		}
		nextUpdate := time.Now().Add(cacheDuration)
		deps.Metrics.SetNextCacheUpdate(nextUpdate)
		logger.Info("Set next cache update time", zap.Time("next_update", nextUpdate))
	}

	// Запускаем обновление кэша если есть данные
	if len(whitelistManager.GetFemaleWhitelist()) > 0 || len(whitelistManager.GetMaleWhitelist()) > 0 {
		logger.Info("Starting cache updater")
		go cache.StartUpdater()
	} else {
		logger.Warn("Cache updater not started due to empty whitelists")
	}

	return bot, nil
}

// createRateLimitMiddleware создает middleware для rate limiting
func createRateLimitMiddleware(rateLimiter middleware.RateLimiterInterface, logger *zap.Logger) types.Middleware {
	return func(ctx types.Context, next types.HandlerFunc) error {
		userID := ctx.Message.From.ID
		if !rateLimiter.AllowRequest(userID) {
			logger.Warn("Rate limit exceeded",
				zap.Int64("user_id", userID),
				zap.String("command", ctx.Message.Command()))
			return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID,
				"Слишком много запросов. Попробуйте позже.")
		}
		return next(ctx)
	}
}

// Start runs the bot
func (b *Bot) Start() error {
	defer b.keyboard.Stop()
	defer b.workerPool.Stop()

	// Запускаем worker pool
	b.workerPool.Start()

	// Запускаем worker pool в keyboard manager
	b.keyboard.StartWorkerPool()

	// Запускаем health check сервер
	if b.health != nil {
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			if err := b.health.Start(); err != nil {
				b.logger.Error("Health check server failed", zap.Error(err))
			}
		}()
	}

	// Запускаем очистку rate limiter
	if b.rateLimiter != nil {
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			ticker := time.NewTicker(b.config.GetRateLimitWindow())
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					b.rateLimiter.Cleanup()
				case <-b.stopChan:
					return
				}
			}
		}()
	}

	reconnectDelay := 10 * time.Second // Задержка между попытками реконнекта

	for {
		select {
		case <-b.stopChan:
			b.logger.Info("Received stop signal before start polling")
			return nil
		default:
		}

		tgAPI := b.api.(*botapi.TelegramBotAPI).GetAPI()
		b.logger.Info("Bot started", zap.String("username", tgAPI.Self.UserName))

		_, err := tgAPI.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
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
		updatesChan := tgAPI.GetUpdatesChan(u)
		if updatesChan == nil {
			b.logger.Error("Failed to create updates channel, will retry after delay")
			time.Sleep(reconnectDelay)
			continue
		}

		for update := range updatesChan {
			select {
			case <-b.stopChan:
				b.logger.Info("Received stop signal")
				return nil
			default:
			}

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
				// Обрабатываем callback query через worker pool
				job := worker.Job{
					UpdateID: update.UpdateID,
					UserID:   update.CallbackQuery.From.ID,
					Command:  "callback_query",
					Handler: func() error {
						b.keyboard.HandleCallbackQuery(update.CallbackQuery)
						return nil
					},
				}
				if err := b.workerPool.Submit(job); err != nil {
					b.logger.Error("Failed to submit callback job", zap.Error(err))
					// Fallback к синхронной обработке
					go b.keyboard.HandleCallbackQuery(update.CallbackQuery)
				}
				continue
			}

			if !update.Message.IsCommand() {
				continue
			}

			// Обрабатываем команды через worker pool
			job := worker.Job{
				UpdateID: update.UpdateID,
				UserID:   update.Message.From.ID,
				Command:  update.Message.Command(),
				Handler: func() error {
					b.handleUpdate(update)
					return nil
				},
			}
			if err := b.workerPool.Submit(job); err != nil {
				b.logger.Error("Failed to submit command job", zap.Error(err))
				// Fallback к синхронной обработке
				go b.handleUpdate(update)
			}
		}

		b.logger.Warn("Update channel closed, will try to reconnect after delay")
		time.Sleep(reconnectDelay)
	}
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

// Stop gracefully stops the bot
func (b *Bot) Stop() error {
	b.logger.Info("Stopping bot gracefully")

	// Отправляем сигнал остановки
	close(b.stopChan)

	// Останавливаем health check сервер
	if b.health != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := b.health.Stop(ctx); err != nil {
			b.logger.Error("Failed to stop health check server", zap.Error(err))
		}
	}

	// Ждем завершения всех горутин
	b.wg.Wait()

	// Останавливаем worker pool
	b.workerPool.Stop()

	// Останавливаем keyboard manager
	b.keyboard.Stop()

	// Очищаем кэш
	b.deps.Cache.Clear()

	// Логируем метрики worker pool
	b.logger.Info("Worker pool metrics",
		zap.Int64("processed_jobs", b.workerPool.GetProcessedJobs()),
		zap.Int64("failed_jobs", b.workerPool.GetFailedJobs()),
		zap.Duration("total_processing_time", b.workerPool.GetProcessingTime()))

	b.logger.Info("Bot stopped successfully")
	return nil
}

// extractMonth extracts the month from callback data
func extractMonth(data string) string {
	if strings.HasPrefix(data, "month_") {
		return strings.TrimPrefix(data, "month_")
	}
	return "unknown"
}
