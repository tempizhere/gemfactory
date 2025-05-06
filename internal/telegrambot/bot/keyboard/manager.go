package keyboard

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/pkg/config"
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
}

// NewKeyboardManager creates a new KeyboardManager instance with cached keyboards
func NewKeyboardManager(api botapi.BotAPI, logger *zap.Logger, al *artistlist.ArtistList, config *config.Config, cache cache.Cache) *KeyboardManager {
	k := &KeyboardManager{
		api:       api,
		logger:    logger,
		debouncer: debounce.NewDebouncer(),
		svc:       service.NewReleaseService(al, config, logger, cache),
		config:    config,
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(release.Months); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3 && i+j < len(release.Months); j++ {
			month := release.Months[i+j]
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(cases.Title(language.English).String(month), "month_"+month))
		}
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Back", "back_to_main")))
	k.allMonthsKeyboard = tgbotapi.NewInlineKeyboardMarkup(rows...)

	k.updateMainMonthKeyboard()

	go func() {
		for {
			// Загружаем таймзону из конфигурации
			loc, err := time.LoadLocation(k.config.Timezone)
			if err != nil {
				k.logger.Error("Failed to load timezone", zap.String("timezone", k.config.Timezone), zap.Error(err))
				loc = time.UTC // Запасная таймзона
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

// updateMainMonthKeyboard updates the main month keyboard with the current, previous, and next months
func (k *KeyboardManager) updateMainMonthKeyboard() {
	// Загружаем таймзону из конфигурации
	loc, err := time.LoadLocation(k.config.Timezone)
	if err != nil {
		k.logger.Error("Failed to load timezone", zap.String("timezone", k.config.Timezone), zap.Error(err))
		loc = time.UTC // Запасная таймзона
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

	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(cases.Title(language.English).String(release.Months[prevMonth-1]), "month_"+release.Months[prevMonth-1]),
		tgbotapi.NewInlineKeyboardButtonData(cases.Title(language.English).String(release.Months[currentMonth-1]), "month_"+release.Months[currentMonth-1]),
		tgbotapi.NewInlineKeyboardButtonData(cases.Title(language.English).String(release.Months[nextMonth-1]), "month_"+release.Months[nextMonth-1]),
		tgbotapi.NewInlineKeyboardButtonData("...", "show_all_months"),
	}

	k.mainMonthKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)
	k.logger.Info("Updated main month keyboard", zap.String("current_month", release.Months[currentMonth-1]), zap.String("timezone", k.config.Timezone))
}

// GetMainKeyboard returns the cached main month keyboard
func (k *KeyboardManager) GetMainKeyboard() tgbotapi.InlineKeyboardMarkup {
	return k.mainMonthKeyboard
}

// GetAllMonthsKeyboard returns the cached all months keyboard
func (k *KeyboardManager) GetAllMonthsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return k.allMonthsKeyboard
}

// HandleCallbackQuery processes callback queries from inline keyboards
func (k *KeyboardManager) HandleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID

	if strings.HasPrefix(data, "month_") {
		debounceKey := fmt.Sprintf("%d:%s", chatID, data)
		if !k.debouncer.CanProcessRequest(debounceKey) {
			k.logger.Info("Callback query debounced", zap.Int64("chat_id", chatID), zap.String("data", data))
			return
		}
	}

	if data == "show_all_months" {
		if err := k.api.EditMessageReplyMarkup(chatID, callback.Message.MessageID, k.GetAllMonthsKeyboard()); err != nil {
			k.logger.Error("Failed to edit message markup", zap.Error(err))
		}
		return
	}

	if data == "back_to_main" {
		if err := k.api.EditMessageReplyMarkup(chatID, callback.Message.MessageID, k.GetMainKeyboard()); err != nil {
			k.logger.Error("Failed to edit message markup", zap.Error(err))
		}
		return
	}

	if strings.HasPrefix(data, "month_") {
		month := strings.TrimPrefix(data, "month_")
		response, err := k.svc.GetReleasesForMonth(month, false, false)
		if err != nil {
			k.logger.Error("Failed to get releases for month", zap.String("month", month), zap.Error(err))
			if err := k.api.SendMessage(chatID, fmt.Sprintf("Ошибка: %v", err)); err != nil {
				k.logger.Error("Failed to send error message", zap.Int64("chat_id", chatID), zap.Error(err))
			}
			return
		}
		if err := k.api.SendMessageWithMarkup(chatID, response, k.GetMainKeyboard()); err != nil {
			k.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.String("text", response), zap.Error(err))
		}
		return
	}

	k.logger.Warn("Unknown callback query", zap.String("data", data))
	if err := k.api.SendMessage(chatID, "Неизвестный запрос."); err != nil {
		k.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.String("text", "Неизвестный запрос."), zap.Error(err))
	}
}

// Stop is a no-op since periodic updates are managed with a simple sleep loop
func (k *KeyboardManager) Stop() {}
