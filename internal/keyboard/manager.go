// Package keyboard реализует менеджер клавиатур для Telegram-бота.
package keyboard

import (
	"fmt"
	"gemfactory/internal/config"
	"gemfactory/internal/external/telegram"
	"gemfactory/internal/service"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Manager реализует менеджер клавиатур для Telegram-бота.
type Manager struct {
	services          *service.Services
	logger            *zap.Logger
	config            *config.Config
	botAPI            telegram.BotAPI
	allMonthsKeyboard tgbotapi.InlineKeyboardMarkup
	mainMonthKeyboard tgbotapi.InlineKeyboardMarkup
	stopChan          chan struct{}
}

var _ ManagerInterface = (*Manager)(nil)

// NewKeyboardManager создает новый менеджер клавиатур
func NewKeyboardManager(services *service.Services, config *config.Config, logger *zap.Logger) *Manager {
	k := &Manager{
		services: services,
		logger:   logger,
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Инициализируем клавиатуры
	k.initKeyboards()

	// Запускаем обновление основной клавиатуры
	go k.updateMainMonthKeyboardLoop()

	return k
}

// SetBotAPI устанавливает BotAPI для отправки сообщений
func (k *Manager) SetBotAPI(botAPI telegram.BotAPI) {
	k.botAPI = botAPI
}

// initKeyboards инициализирует все клавиатуры
func (k *Manager) initKeyboards() {
	// Создаем клавиатуру со всеми месяцами
	k.initAllMonthsKeyboard()

	// Создаем основную клавиатуру
	k.updateMainMonthKeyboard()
}

// initAllMonthsKeyboard создает клавиатуру со всеми месяцами
func (k *Manager) initAllMonthsKeyboard() {
	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(months); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3 && i+j < len(months); j++ {
			month := months[i+j]
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(
				cases.Title(language.English).String(month),
				"month_"+month,
			))
		}
		rows = append(rows, row)
	}

	// Добавляем кнопку "Назад"
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "back_to_main"),
	))

	k.allMonthsKeyboard = tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// updateMainMonthKeyboard обновляет основную клавиатуру с текущим месяцем
func (k *Manager) updateMainMonthKeyboard() {
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

	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			cases.Title(language.English).String(months[prevMonth-1]),
			"month_"+months[prevMonth-1],
		),
		tgbotapi.NewInlineKeyboardButtonData(
			cases.Title(language.English).String(months[currentMonth-1]),
			"month_"+months[currentMonth-1],
		),
		tgbotapi.NewInlineKeyboardButtonData(
			cases.Title(language.English).String(months[nextMonth-1]),
			"month_"+months[nextMonth-1],
		),
		tgbotapi.NewInlineKeyboardButtonData("...", "show_all_months"),
	}

	k.mainMonthKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)

	k.logger.Info("Updated main month keyboard", zap.String("current_month", months[currentMonth-1]))
}

// updateMainMonthKeyboardLoop запускает цикл обновления основной клавиатуры
func (k *Manager) updateMainMonthKeyboardLoop() {
	for {
		select {
		case <-k.stopChan:
			return
		default:
			loc, err := time.LoadLocation(k.config.Timezone)
			if err != nil {
				k.logger.Error("Failed to load timezone", zap.String("timezone", k.config.Timezone), zap.Error(err))
				loc = time.UTC
			}

			now := time.Now().In(loc)
			nextMonth := now.AddDate(0, 1, 0)
			firstOfNextMonth := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, loc)
			durationUntilFirst := firstOfNextMonth.Sub(now)

			select {
			case <-time.After(durationUntilFirst):
				k.updateMainMonthKeyboard()
			case <-k.stopChan:
				return
			}
		}
	}
}

// GetMainKeyboard возвращает основную клавиатуру
func (k *Manager) GetMainKeyboard() tgbotapi.InlineKeyboardMarkup {
	return k.mainMonthKeyboard
}

// GetAllMonthsKeyboard возвращает клавиатуру со всеми месяцами
func (k *Manager) GetAllMonthsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return k.allMonthsKeyboard
}

// HandleCallbackQuery обрабатывает callback query от inline клавиатур
func (k *Manager) HandleCallbackQuery(callback *tgbotapi.CallbackQuery) error {
	data := callback.Data
	chatID := callback.Message.Chat.ID

	k.logger.Debug("Received callback query", zap.String("data", data), zap.Int64("chat_id", chatID))

	if strings.HasPrefix(data, "month_") {
		return k.handleMonthCallback(callback)
	}

	if data == "show_all_months" {
		return k.handleShowAllMonthsCallback(callback)
	}

	if data == "back_to_main" {
		return k.handleBackToMainCallback(callback)
	}

	k.logger.Warn("Unknown callback query", zap.String("data", data))
	return fmt.Errorf("unknown callback query: %s", data)
}

// handleMonthCallback обрабатывает callback для выбора месяца
func (k *Manager) handleMonthCallback(callback *tgbotapi.CallbackQuery) error {
	data := callback.Data
	chatID := callback.Message.Chat.ID

	month := strings.TrimPrefix(data, "month_")
	k.logger.Debug("Processing month callback", zap.String("month", month))

	// Получаем релизы за месяц
	response, err := k.services.Release.GetReleasesForMonth(month, false, false)
	if err != nil {
		k.logger.Error("Failed to get releases for month", zap.String("month", month), zap.Error(err))
		return fmt.Errorf("failed to get releases for month %s: %w", month, err)
	}

	if response == "" {
		k.logger.Warn("Empty response for month", zap.String("month", month))
		response = fmt.Sprintf("Релизы для %s не найдены.", month)
	}

	// Отправляем ответ с основной клавиатурой
	msg := tgbotapi.NewMessage(chatID, response)
	msg.ReplyMarkup = k.GetMainKeyboard()

	// Отправляем сообщение через BotAPI
	if k.botAPI != nil {
		err := k.botAPI.SendMessageWithMarkup(chatID, response, msg.ReplyMarkup)
		if err != nil {
			k.logger.Error("Failed to send message with markup", zap.Int64("chat_id", chatID), zap.Error(err))
			return err
		}
	} else {
		k.logger.Warn("BotAPI not available, cannot send message", zap.Int64("chat_id", chatID))
	}

	return nil
}

// handleShowAllMonthsCallback обрабатывает callback для показа всех месяцев
func (k *Manager) handleShowAllMonthsCallback(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	k.logger.Debug("Showing all months keyboard")

	// Редактируем сообщение через BotAPI
	if k.botAPI != nil {
		err := k.botAPI.EditMessageReplyMarkup(chatID, messageID, k.GetAllMonthsKeyboard())
		if err != nil {
			k.logger.Error("Failed to edit message markup", zap.Int64("chat_id", chatID), zap.Error(err))
			return err
		}
	} else {
		k.logger.Warn("BotAPI not available, cannot edit message", zap.Int64("chat_id", chatID))
	}

	return nil
}

// handleBackToMainCallback обрабатывает callback для возврата к основной клавиатуре
func (k *Manager) handleBackToMainCallback(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	k.logger.Debug("Returning to main keyboard")

	// Редактируем сообщение через BotAPI
	if k.botAPI != nil {
		err := k.botAPI.EditMessageReplyMarkup(chatID, messageID, k.GetMainKeyboard())
		if err != nil {
			k.logger.Error("Failed to edit message markup", zap.Int64("chat_id", chatID), zap.Error(err))
			return err
		}
	} else {
		k.logger.Warn("BotAPI not available, cannot edit message", zap.Int64("chat_id", chatID))
	}

	return nil
}

// Stop останавливает менеджер клавиатур
func (k *Manager) Stop() {
	close(k.stopChan)
}
