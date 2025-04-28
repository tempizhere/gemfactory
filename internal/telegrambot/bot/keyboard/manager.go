package keyboard

import (
	"fmt"
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/releasefmt"
	"gemfactory/pkg/config"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
	"time"
)

// KeyboardManager manages Inline Keyboards for the bot
type KeyboardManager struct {
	mainMonthKeyboard tgbotapi.InlineKeyboardMarkup
	allMonthsKeyboard tgbotapi.InlineKeyboardMarkup
	api               botapi.BotAPI // Используем интерфейс BotAPI
	logger            *zap.Logger
	debouncer         *debounce.Debouncer
	al                *artistlist.ArtistList
	config            *config.Config
}

// NewKeyboardManager creates a new KeyboardManager instance with cached keyboards
func NewKeyboardManager(api botapi.BotAPI, logger *zap.Logger, al *artistlist.ArtistList, config *config.Config) *KeyboardManager {
	k := &KeyboardManager{
		api:       api,
		logger:    logger,
		debouncer: debounce.NewDebouncer(),
		al:        al,
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
			now := time.Now()
			nextMonth := now.AddDate(0, 1, 0)
			firstOfNextMonth := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location())
			durationUntilFirst := firstOfNextMonth.Sub(now)
			time.Sleep(durationUntilFirst)
			k.updateMainMonthKeyboard()
		}
	}()

	return k
}

// updateMainMonthKeyboard updates the main month keyboard with the current, previous, and next months
func (k *KeyboardManager) updateMainMonthKeyboard() {
	currentMonth := int(time.Now().Month())
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
	k.logger.Info("Updated main month keyboard", zap.String("current_month", release.Months[currentMonth-1]))
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
		whitelist := k.al.GetUnitedWhitelist()
		releases, err := cache.GetReleasesForMonths([]string{month}, whitelist, false, false, k.al, k.config, k.logger)
		if err != nil {
			if err := k.api.SendMessage(chatID, fmt.Sprintf("Ошибка при получении релизов: %v", err)); err != nil {
				k.logger.Error("Failed to send message", zap.Error(err))
			}
			return
		}

		if len(releases) == 0 {
			if err := k.api.SendMessage(chatID, "Релизы не найдены."); err != nil {
				k.logger.Error("Failed to send message", zap.Error(err))
			}
			return
		}

		var response strings.Builder
		for _, rel := range releases {
			formatted := releasefmt.FormatReleaseForTelegram(rel, k.logger)
			response.WriteString(formatted + "\n")
		}

		if err := k.api.SendMessageWithMarkup(chatID, response.String(), k.GetMainKeyboard()); err != nil {
			k.logger.Error("Failed to send message", zap.Error(err))
		}
	}
}

// Stop is a no-op since periodic updates are managed with a simple sleep loop
func (k *KeyboardManager) Stop() {}
