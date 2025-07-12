package keyboard

import (
	"fmt"
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/worker"
	"gemfactory/internal/telegrambot/releases/artist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/pkg/config"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// KeyboardManager manages Inline Keyboards for the bot
type KeyboardManager struct {
	mainMonthKeyboard tgbotapi.InlineKeyboardMarkup
	allMonthsKeyboard tgbotapi.InlineKeyboardMarkup
	api               botapi.BotAPI
	logger            *zap.Logger
	debouncer         *debounce.Debouncer
	svc               *service.ReleaseService
	config            *config.Config
	workerPool        worker.WorkerPoolInterface
}

// Убеждаемся, что KeyboardManager реализует KeyboardManagerInterface
var _ KeyboardManagerInterface = (*KeyboardManager)(nil)

// NewKeyboardManager creates a new KeyboardManager instance with cached keyboards
func NewKeyboardManager(api botapi.BotAPI, logger *zap.Logger, al artist.WhitelistManager, config *config.Config, cache cache.Cache) *KeyboardManager {
	k := &KeyboardManager{
		api:        api,
		logger:     logger,
		debouncer:  debounce.NewDebouncer(),
		svc:        service.NewReleaseService(al, config, logger, cache),
		config:     config,
		workerPool: worker.NewWorkerPool(config.MaxConcurrentRequests, 50, logger),
	}

	cfg := release.NewConfig()
	months := cfg.Months()
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(months); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3 && i+j < len(months); j++ {
			month := months[i+j]
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(cases.Title(language.English).String(month), "month_"+month))
		}
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Back", "back_to_main")))
	k.allMonthsKeyboard = tgbotapi.NewInlineKeyboardMarkup(rows...)

	k.updateMainMonthKeyboard()

	// Запускаем worker pool
	k.workerPool.Start()

	go func() {
		for {
			loc, err := time.LoadLocation(k.config.Timezone)
			if err != nil {
				k.logger.Error("Failed to load timezone", zap.String("timezone", k.config.Timezone), zap.Error(err))
				loc = time.UTC
			}
			now := time.Now().In(loc)
			nextMonth := now.AddDate(0, 1, 0)
			firstOfNextMonth := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, loc)
			durationUntilFirst := firstOfNextMonth.Sub(now)
			time.Sleep(durationUntilFirst)
			k.updateMainMonthKeyboard()
		}
	}()

	return k
}

// updateMainMonthKeyboard updates the main month keyboard
func (k *KeyboardManager) updateMainMonthKeyboard() {
	loc, err := time.LoadLocation(k.config.Timezone)
	if err != nil {
		k.logger.Error("Failed to load timezone", zap.String("timezone", k.config.Timezone), zap.Error(err))
		loc = time.UTC
	}
	currentMonth := int(time.Now().In(loc).Month())
	prevMonth := currentMonth - 1
	if prevMonth < 1 {
		prevMonth = 12
	}
	nextMonth := currentMonth + 1
	if nextMonth > 12 {
		nextMonth = 1
	}

	cfg := release.NewConfig()
	months := cfg.Months()
	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(cases.Title(language.English).String(months[prevMonth-1]), "month_"+months[prevMonth-1]),
		tgbotapi.NewInlineKeyboardButtonData(cases.Title(language.English).String(months[currentMonth-1]), "month_"+months[currentMonth-1]),
		tgbotapi.NewInlineKeyboardButtonData(cases.Title(language.English).String(months[nextMonth-1]), "month_"+months[nextMonth-1]),
		tgbotapi.NewInlineKeyboardButtonData("...", "show_all_months"),
	}

	k.mainMonthKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)
	k.logger.Info("Updated main month keyboard", zap.String("current_month", months[currentMonth-1]))
}

// GetMainKeyboard returns the main month keyboard
func (k *KeyboardManager) GetMainKeyboard() tgbotapi.InlineKeyboardMarkup {
	return k.mainMonthKeyboard
}

// GetAllMonthsKeyboard returns the all months keyboard
func (k *KeyboardManager) GetAllMonthsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return k.allMonthsKeyboard
}

// StartWorkerPool запускает worker pool для keyboard manager
func (k *KeyboardManager) StartWorkerPool() {
	k.workerPool.Start()
}

// StopWorkerPool останавливает worker pool для keyboard manager
func (k *KeyboardManager) StopWorkerPool() {
	k.workerPool.Stop()
}

// HandleCallbackQuery processes callback queries from inline keyboards using worker pool
func (k *KeyboardManager) HandleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID

	k.logger.Debug("Received callback query", zap.String("data", data), zap.Int64("chat_id", chatID))

	// Создаем задачу для обработки callback query
	job := worker.Job{
		UpdateID: 0, // Не используется для keyboard
		UserID:   callback.From.ID,
		Command:  "callback_query",
		Handler: func() error {
			return k.processCallbackQuery(callback)
		},
	}

	if err := k.workerPool.Submit(job); err != nil {
		k.logger.Error("Failed to submit callback query job", zap.Error(err))
		// Fallback к синхронной обработке
		k.processCallbackQuery(callback)
	}
}

// processCallbackQuery обрабатывает callback query синхронно
func (k *KeyboardManager) processCallbackQuery(callback *tgbotapi.CallbackQuery) error {
	data := callback.Data
	chatID := callback.Message.Chat.ID

	if strings.HasPrefix(data, "month_") {
		debounceKey := fmt.Sprintf("%d:%s", chatID, data)
		if !k.debouncer.CanProcessRequest(debounceKey) {
			k.logger.Info("Callback query debounced", zap.Int64("chat_id", chatID), zap.String("data", data))
			return nil
		}

		month := strings.TrimPrefix(data, "month_")
		k.logger.Debug("Processing month callback", zap.String("month", month))

		response, err := k.svc.GetReleasesForMonth(month, false, false)
		if err != nil {
			k.logger.Error("Failed to get releases for month", zap.String("month", month), zap.Error(err))
			// Use the error message directly for user-friendly output
			if err := k.api.SendMessage(chatID, err.Error()); err != nil {
				k.logger.Error("Failed to send error message", zap.Int64("chat_id", chatID), zap.Error(err))
			}
			return err
		}

		if response == "" {
			k.logger.Warn("Empty response for month", zap.String("month", month))
			if err := k.api.SendMessage(chatID, fmt.Sprintf("Релизы для %s не найдены.", month)); err != nil {
				k.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.Error(err))
			}
			return nil
		}

		k.logger.Debug("Sending releases for month", zap.String("month", month), zap.String("response", response))
		if err := k.api.SendMessageWithMarkup(chatID, response, k.GetMainKeyboard()); err != nil {
			k.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.String("text", response), zap.Error(err))
			return err
		}
		return nil
	}

	if data == "show_all_months" {
		k.logger.Debug("Showing all months keyboard")
		if err := k.api.EditMessageReplyMarkup(chatID, callback.Message.MessageID, k.GetAllMonthsKeyboard()); err != nil {
			k.logger.Error("Failed to edit message markup", zap.Error(err))
			return err
		}
		return nil
	}

	if data == "back_to_main" {
		k.logger.Debug("Returning to main keyboard")
		if err := k.api.EditMessageReplyMarkup(chatID, callback.Message.MessageID, k.GetMainKeyboard()); err != nil {
			k.logger.Error("Failed to edit message markup", zap.Error(err))
			return err
		}
		return nil
	}

	k.logger.Warn("Unknown callback query", zap.String("data", data))
	if err := k.api.SendMessage(chatID, "Неизвестный запрос."); err != nil {
		k.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.Error(err))
		return err
	}
	return nil
}

// Stop stops the keyboard manager
func (k *KeyboardManager) Stop() {
	k.StopWorkerPool()

	// Логируем метрики worker pool
	k.logger.Info("Keyboard manager worker pool metrics",
		zap.Int64("processed_jobs", k.workerPool.GetProcessedJobs()),
		zap.Int64("failed_jobs", k.workerPool.GetFailedJobs()),
		zap.Duration("total_processing_time", k.workerPool.GetProcessingTime()))
}
