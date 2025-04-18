package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// HandleCallbackQuery handles Inline Keyboard button presses
func (h *CommandHandlers) HandleCallbackQuery(update tgbotapi.Update) {
	callback := update.CallbackQuery
	chatID := callback.Message.Chat.ID
	username := callback.From.UserName
	callbackData := callback.Data

	// Проверяем дабл-клик только для кнопок с названиями месяцев
	shouldDebounce := strings.HasPrefix(callbackData, "month_")
	if shouldDebounce {
		// Формируем ключ для проверки дабл-клика: chatID + месяц (например, "123-month_april")
		// Это гарантирует, что дабл-клик срабатывает только для повторных нажатий на один и тот же месяц
		requestKey := fmt.Sprintf("%d-%s", chatID, callbackData)
		if !h.debouncer.CanProcessRequest(requestKey) {
			// Игнорируем повторный запрос, но подтверждаем callback
			callbackConfig := tgbotapi.NewCallback(callback.ID, "")
			h.api.Send(callbackConfig)
			return
		}
	}

	if callbackData == "show_all_months" {
		// Показываем все 12 месяцев
		msg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Выберите месяц:")
		msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{}
		*msg.ReplyMarkup = h.keyboard.GetAllMonthsKeyboard()
		h.api.Send(msg)

		// Подтверждаем callback
		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.api.Send(callbackConfig)
		return
	}

	if callbackData == "back_to_main" {
		// Возвращаемся к основному меню с текущим, предыдущим и следующим месяцем
		msg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Выберите месяц:")
		msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{}
		*msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
		h.api.Send(msg)

		// Подтверждаем callback
		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.api.Send(callbackConfig)
		return
	}

	if strings.HasPrefix(callbackData, "month_") {
		month := strings.TrimPrefix(callbackData, "month_")

		// Логируем выбор месяца
		h.logger.Info("User selected month", zap.String("username", username), zap.String("month", month))

		// Выполняем логику команды /month <month> асинхронно
		go func() {
			months := []string{month}
			releases, err := h.fetchReleases(months, false, false, chatID)
			if err != nil {
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка получения релизов: %v", err))
				msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
				h.api.Send(msg)
				return
			}

			if len(releases) == 0 {
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Релизы для %s не найдены. Проверьте whitelist или данные на сайте.", month))
				msg.ParseMode = "HTML"
				msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
				h.api.Send(msg)
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
			msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
			h.api.Send(msg)

			// Подтверждаем callback
			callbackConfig := tgbotapi.NewCallback(callback.ID, "")
			h.api.Send(callbackConfig)
		}()
	}
}
