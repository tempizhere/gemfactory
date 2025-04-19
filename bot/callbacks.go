package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// HandleCallbackQuery handles Inline Keyboard button presses
func (h *CommandHandlers) HandleCallbackQuery(update tgbotapi.Update) {
	if update.CallbackQuery == nil {
		h.logger.Error("CallbackQuery is nil in HandleCallbackQuery")
		return
	}

	callback := update.CallbackQuery
	chatID := callback.Message.Chat.ID
	callbackData := callback.Data

	// Проверяем дабл-клик только для кнопок с названиями месяцев
	shouldDebounce := strings.HasPrefix(callbackData, "month_")
	if shouldDebounce {
		// Формируем ключ для проверки дабл-клика
		requestKey := h.buildDebounceKey(chatID, callbackData)
		if !h.debouncer.CanProcessRequest(requestKey) {
			h.confirmCallback(callback.ID)
			return
		}
	}

	if callbackData == "show_all_months" {
		// Показываем все 12 месяцев
		msg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Выберите месяц:")
		h.editMessageWithKeyboard(msg, h.keyboard.GetAllMonthsKeyboard())
		h.confirmCallback(callback.ID)
		return
	}

	if callbackData == "back_to_main" {
		// Возвращаемся к основному меню с текущим, предыдущим и следующим месяцем
		msg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Выберите месяц:")
		h.editMessageWithKeyboard(msg, h.keyboard.GetMainMonthKeyboard())
		h.confirmCallback(callback.ID)
		return
	}

	if strings.HasPrefix(callbackData, "month_") {
		month := strings.TrimPrefix(callbackData, "month_")

		// Выполняем логику команды /month <month> асинхронно
		go func() {
			months := []string{month}
			releases, err := h.fetchReleases(months, false, false, chatID)
			if err != nil {
				h.logger.Error("Failed to fetch releases", zap.Error(err))
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка получения релизов: %v", err))
				h.sendMessageWithKeyboard(msg, h.keyboard.GetMainMonthKeyboard())
				return
			}

			if len(releases) == 0 {
				h.logger.Info("No releases found for month", zap.String("month", month))
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Релизы для %s не найдены. Проверьте whitelist или данные на сайте.", month))
				msg.ParseMode = "HTML"
				h.sendMessageWithKeyboard(msg, h.keyboard.GetMainMonthKeyboard())
				return
			}

			// Форматируем ответ
			var response strings.Builder
			for _, release := range releases {
				response.WriteString(formatReleaseForTelegram(release))
				response.WriteString("\n")
			}

			msg := tgbotapi.NewMessage(chatID, response.String())
			msg.ParseMode = "HTML"
			msg.DisableWebPagePreview = true
			h.sendMessageWithKeyboard(msg, h.keyboard.GetMainMonthKeyboard())

			// Подтверждаем callback
			h.confirmCallback(callback.ID)
		}()
	}
}

// sendMessageWithKeyboard sends a new message with the specified keyboard
func (h *CommandHandlers) sendMessageWithKeyboard(msg tgbotapi.MessageConfig, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg.ReplyMarkup = keyboard
	if _, err := h.api.Send(msg); err != nil {
		h.logger.Error("Failed to send message", zap.String("message", msg.Text), zap.Error(err))
	}
}

// editMessageWithKeyboard edits an existing message with the specified keyboard
func (h *CommandHandlers) editMessageWithKeyboard(msg tgbotapi.EditMessageTextConfig, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg.ReplyMarkup = &keyboard
	if _, err := h.api.Send(msg); err != nil {
		h.logger.Error("Failed to edit message", zap.String("message", msg.Text), zap.Error(err))
	}
}

// confirmCallback confirms the callback query to Telegram
func (h *CommandHandlers) confirmCallback(callbackID string) {
	callbackConfig := tgbotapi.NewCallback(callbackID, "")
	if _, err := h.api.Request(callbackConfig); err != nil {
		h.logger.Error("Failed to send callback confirmation", zap.Error(err))
	}
}

// buildDebounceKey creates a debounce key for double-click prevention
func (h *CommandHandlers) buildDebounceKey(chatID int64, callbackData string) string {
	return fmt.Sprintf("%d-%s", chatID, callbackData)
}
