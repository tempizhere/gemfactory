// Package bot содержит основную логику Telegram-бота.
package bot

import (
	"context"
	"fmt"
	commands "gemfactory/internal/bot/handlers"
	"gemfactory/internal/bot/keyboard"
	"gemfactory/internal/bot/middleware"
	"gemfactory/internal/bot/router"
	"gemfactory/internal/config"
	"gemfactory/internal/domain/types"
	"gemfactory/internal/gateway/telegram/botapi"
	"gemfactory/internal/infrastructure/health"
	"gemfactory/internal/infrastructure/worker"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Bot represents the main bot instance
type Bot struct {
	api         botapi.BotAPI
	logger      *zap.Logger
	config      config.Interface
	router      router.Interface
	deps        *types.Dependencies
	keyboard    keyboard.ManagerInterface
	workerPool  worker.PoolInterface
	health      health.ServerInterface
	rateLimiter middleware.RateLimiterInterface
	stopChan    chan struct{}
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

var _ types.Interface = (*Bot)(nil)

// NewBot creates a new Bot instance
func NewBot(config *config.Config, logger *zap.Logger) (*Bot, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	factory := NewComponentFactory(config, logger)

	// Валидируем конфигурацию
	if err := factory.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Создаем все зависимости через фабрику
	deps, err := factory.CreateDependencies()
	if err != nil {
		return nil, fmt.Errorf("failed to create dependencies: %w", err)
	}

	// Проверяем наличие артистов в whitelist
	whitelistManager := factory.CreateWhitelistManager()
	if len(whitelistManager.GetFemaleWhitelist()) == 0 && len(whitelistManager.GetMaleWhitelist()) == 0 {
		logger.Warn("Both female and male whitelists are empty; populate at least one whitelist using /add_artist")
	}

	// Создаем дополнительные компоненты
	rateLimiter := factory.CreateRateLimiter()
	healthServer := factory.CreateServer(deps.BotAPI, deps.Cache, deps.WorkerPool)

	// Настраиваем роутер
	r := router.NewRouter()
	r.Use(middleware.LogRequest)
	r.Use(middleware.MetricsMiddleware)
	r.Use(middleware.ErrorHandler)

	// Добавляем rate limiting middleware если включен
	if rateLimiter != nil {
		r.Use(createRateLimitMiddleware(rateLimiter, logger))
	}

	logger.Info("Initializing command routes")
	commands.RegisterRoutes(r, deps)

	// Создаем контекст для управления жизненным циклом
	ctx, cancel := context.WithCancel(context.Background())

	bot := &Bot{
		api:         deps.BotAPI,
		logger:      logger,
		config:      config,
		router:      r,
		deps:        deps,
		keyboard:    deps.Keyboard,
		workerPool:  deps.WorkerPool, // Используем worker pool из dependencies
		health:      healthServer,
		rateLimiter: rateLimiter,
		stopChan:    make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Запускаем обновление кэша если есть данные
	if len(whitelistManager.GetFemaleWhitelist()) > 0 || len(whitelistManager.GetMaleWhitelist()) > 0 {
		logger.Info("Starting cache updater")
		go deps.Cache.StartUpdater(ctx)
	} else {
		logger.Warn("Cache updater not started due to empty whitelists")
	}

	logger.Info("Bot created successfully")
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
	defer func() {
		b.keyboard.Stop()
		b.workerPool.Stop()
	}()

	// Запускаем worker pool
	b.workerPool.Start()

	// Запускаем health check сервер с контекстом
	if b.health != nil {
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			select {
			case <-b.ctx.Done():
				b.logger.Info("Health check server cancelled by context")
				return
			default:
				if err := b.health.Start(); err != nil {
					// Проверяем, является ли ошибка нормальной остановкой
					if err.Error() == "http: Server closed" {
						b.logger.Info("Health check server stopped normally")
					} else {
						b.logger.Error("Health check server failed", zap.Error(err))
					}
				}
			}
		}()
	}

	// Запускаем очистку rate limiter с контекстом
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
				case <-b.ctx.Done():
					b.logger.Info("Rate limiter cleanup stopped by context")
					return
				case <-b.stopChan:
					b.logger.Info("Rate limiter cleanup stopped by stop signal")
					return
				}
			}
		}()
	}

	b.logger.Info("Bot started successfully")

	// Основной цикл обработки обновлений с контекстом
	for {
		select {
		case <-b.ctx.Done():
			b.logger.Info("Bot main loop cancelled by context")
			return b.ctx.Err()
		case <-b.stopChan:
			b.logger.Info("Bot main loop stopped by stop signal")
			return nil
		default:
			if err := b.runUpdateLoop(); err != nil {
				// Проверяем, является ли ошибка нормальной остановкой
				if err.Error() == "context canceled" || err == context.Canceled {
					b.logger.Info("Update loop stopped due to context cancellation")
					return err
				}
				b.logger.Error("Update loop error", zap.Error(err))
				// При ошибке ждем перед перезапуском
				select {
				case <-b.ctx.Done():
					return b.ctx.Err()
				case <-time.After(10 * time.Second):
					continue
				}
			}
		}
	}
}

// runUpdateLoop запускает цикл обработки обновлений
func (b *Bot) runUpdateLoop() error {
	b.logger.Info("Starting update channel")

	api := b.api.(*botapi.TelegramBotAPI).GetAPI()

	// Инициализация бота
	b.logger.Info("Bot started", zap.String("username", api.Self.UserName))

	_, err := api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	if err != nil {
		b.logger.Error("Failed to delete webhook", zap.Error(err))
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	// Устанавливаем команды бота
	if err := b.deps.SetBotCommands(); err != nil {
		b.logger.Error("Failed to set bot commands", zap.Error(err))
		return fmt.Errorf("failed to set bot commands: %w", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}

	b.logger.Info("Starting to fetch updates")
	updatesChan := api.GetUpdatesChan(u)
	if updatesChan == nil {
		return fmt.Errorf("failed to create updates channel")
	}

	reconnectDelay := 10 * time.Second // Задержка между попытками реконнекта

	for {
		select {
		case <-b.ctx.Done():
			b.logger.Info("Update loop cancelled by context")
			return b.ctx.Err()
		case <-b.stopChan:
			b.logger.Info("Update loop stopped by stop signal")
			return nil
		case update, ok := <-updatesChan:
			if !ok {
				b.logger.Warn("Update channel closed, will try to reconnect after delay")
				select {
				case <-b.ctx.Done():
					return b.ctx.Err()
				case <-time.After(reconnectDelay):
					return fmt.Errorf("update channel closed, reconnecting")
				}
			}

			// Обработка обновления
			b.processUpdate(update)
		}
	}
}

// processUpdate обрабатывает одно обновление
func (b *Bot) processUpdate(update tgbotapi.Update) {
	// Проверяем контекст перед обработкой
	select {
	case <-b.ctx.Done():
		b.logger.Debug("Skipping update processing due to context cancellation")
		return
	default:
	}

	// Улучшенное логирование с helper функциями
	b.logger.Debug("Processing update",
		zap.Int("update_id", update.UpdateID),
		zap.Int64("user_id", getUserID(update)),
		zap.String("command", extractCommand(update)),
		zap.String("update_type", getUpdateType(update)),
	)

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
		return
	}

	if update.CallbackQuery != nil {
		// Обрабатываем callback query через worker pool с контекстом
		job := worker.Job{
			UpdateID: update.UpdateID,
			UserID:   update.CallbackQuery.From.ID,
			Command:  "callback_query",
			Handler: func() error {
				// Проверяем контекст в обработчике
				select {
				case <-b.ctx.Done():
					return b.ctx.Err()
				default:
					b.keyboard.HandleCallbackQuery(update.CallbackQuery)
					return nil
				}
			},
		}
		if err := b.workerPool.Submit(job); err != nil {
			b.logger.Error("Failed to submit callback job", zap.Error(err))
			// Fallback к синхронной обработке
			go func() {
				select {
				case <-b.ctx.Done():
					return
				default:
					b.keyboard.HandleCallbackQuery(update.CallbackQuery)
				}
			}()
		}
		return
	}

	// Обрабатываем команды и вложения файлов
	if !update.Message.IsCommand() && update.Message.Document == nil {
		return
	}

	// Обрабатываем команды через worker pool с контекстом
	job := worker.Job{
		UpdateID: update.UpdateID,
		UserID:   update.Message.From.ID,
		Command:  update.Message.Command(),
		Handler: func() error {
			// Проверяем контекст в обработчике
			select {
			case <-b.ctx.Done():
				return b.ctx.Err()
			default:
				b.handleUpdate(update)
				return nil
			}
		},
	}
	if err := b.workerPool.Submit(job); err != nil {
		b.logger.Error("Failed to submit command job", zap.Error(err))
		// Fallback к синхронной обработке
		go func() {
			select {
			case <-b.ctx.Done():
				return
			default:
				b.handleUpdate(update)
			}
		}()
	}
}

// handleUpdate processes incoming updates
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	ctx := types.Context{
		Message:  update.Message,
		UpdateID: update.UpdateID,
		Deps:     b.deps,
	}

	// Обрабатываем вложения файлов
	if update.Message.Document != nil {
		b.handleDocument(ctx)
		return
	}

	// Обрабатываем команды
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

	// Отменяем контекст для остановки всех горутин
	if b.cancel != nil {
		b.cancel()
	}

	// Отправляем сигнал остановки (для обратной совместимости)
	select {
	case <-b.stopChan:
		// Канал уже закрыт
	default:
		close(b.stopChan)
	}

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), b.config.GetGracefulShutdownTimeout())
	defer shutdownCancel()

	// Останавливаем health check сервер с контекстом
	if b.health != nil {
		go func() {
			if err := b.health.Stop(shutdownCtx); err != nil {
				b.logger.Error("Failed to stop health check server", zap.Error(err))
			}
		}()
	}

	// Ждем завершения всех горутин с таймаутом
	done := make(chan struct{})
	go func() {
		defer close(done)
		b.wg.Wait()
	}()

	select {
	case <-done:
		b.logger.Info("All goroutines stopped successfully")
	case <-shutdownCtx.Done():
		b.logger.Warn("Graceful shutdown timeout exceeded, forcing stop")
	}

	// Останавливаем worker pool
	b.workerPool.Stop()

	// Останавливаем keyboard manager
	b.keyboard.Stop()

	// Очищаем кэш
	if b.deps.Cache != nil {
		b.deps.Cache.Clear()
	}

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

// getUserID извлекает ID пользователя из обновления
func getUserID(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.From.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID
	}
	return 0
}

// handleDocument обрабатывает вложения файлов
func (b *Bot) handleDocument(ctx types.Context) {
	document := ctx.Message.Document

	// Проверяем, что это CSV файл
	if !strings.HasSuffix(strings.ToLower(document.FileName), ".csv") {
		if err := b.api.SendMessage(ctx.Message.Chat.ID, "❌ Пожалуйста, отправьте CSV файл."); err != nil {
			b.logger.Error("Failed to send message", zap.Error(err))
		}
		return
	}

	// Проверяем права администратора
	if ctx.Message.From.UserName != b.config.GetAdminUsername() {
		if err := b.api.SendMessage(ctx.Message.Chat.ID, "❌ Только администратор может загружать плейлисты."); err != nil {
			b.logger.Error("Failed to send message", zap.Error(err))
		}
		return
	}

	// Получаем информацию о файле
	file, err := b.api.GetFile(document.FileID)
	if err != nil {
		b.logger.Error("Failed to get file info", zap.Error(err))
		if err := b.api.SendMessage(ctx.Message.Chat.ID, "❌ Ошибка при получении файла."); err != nil {
			b.logger.Error("Failed to send message", zap.Error(err))
		}
		return
	}

	// Скачиваем файл
	fileURL := file.Link(b.api.(*botapi.TelegramBotAPI).GetAPI().Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		b.logger.Error("Failed to download file", zap.Error(err))
		if err := b.api.SendMessage(ctx.Message.Chat.ID, "❌ Ошибка при скачивании файла."); err != nil {
			b.logger.Error("Failed to send message", zap.Error(err))
		}
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			b.logger.Error("Failed to close response body", zap.Error(err))
		}
	}()

	// Создаем временный файл
	tempFile, err := os.CreateTemp("", "playlist_*.csv")
	if err != nil {
		b.logger.Error("Failed to create temp file", zap.Error(err))
		if err := b.api.SendMessage(ctx.Message.Chat.ID, "❌ Ошибка при создании временного файла."); err != nil {
			b.logger.Error("Failed to send message", zap.Error(err))
		}
		return
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			b.logger.Error("Failed to remove temp file", zap.Error(err))
		}
	}()
	defer func() {
		if err := tempFile.Close(); err != nil {
			b.logger.Error("Failed to close temp file", zap.Error(err))
		}
	}()

	// Копируем содержимое файла
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		b.logger.Error("Failed to copy file content", zap.Error(err))
		if err := b.api.SendMessage(ctx.Message.Chat.ID, "❌ Ошибка при копировании файла."); err != nil {
			b.logger.Error("Failed to send message", zap.Error(err))
		}
		return
	}

	// Загружаем плейлист
	b.deps.PlaylistManager.Clear()
	if err := b.deps.PlaylistManager.LoadPlaylistFromFile(tempFile.Name()); err != nil {
		b.logger.Error("Failed to load playlist", zap.Error(err))
		if err := b.api.SendMessage(ctx.Message.Chat.ID, fmt.Sprintf("❌ Ошибка при загрузке плейлиста: %v", err)); err != nil {
			b.logger.Error("Failed to send message", zap.Error(err))
		}
		return
	}

	// Плейлист автоматически сохраняется в постоянное хранилище при загрузке
	trackCount := b.deps.PlaylistManager.GetTotalTracks()
	if err := b.api.SendMessage(ctx.Message.Chat.ID,
		fmt.Sprintf("✅ Плейлист успешно загружен и сохранен! Загружено %d треков из файла: %s", trackCount, document.FileName)); err != nil {
		b.logger.Error("Failed to send message", zap.Error(err))
	}
}

// extractCommand извлекает команду из обновления
func extractCommand(update tgbotapi.Update) string {
	if update.Message != nil && update.Message.IsCommand() {
		return update.Message.Command()
	}
	if update.CallbackQuery != nil {
		return "callback"
	}
	return ""
}

// getUpdateType определяет тип обновления
func getUpdateType(update tgbotapi.Update) string {
	if update.Message != nil {
		if update.Message.IsCommand() {
			return "command"
		}
		return "message"
	}
	if update.CallbackQuery != nil {
		return "callback"
	}
	return "unknown"
}
