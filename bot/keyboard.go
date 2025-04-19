package bot

import (
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// months defines the list of month names
var months = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

// KeyboardHandler manages Inline Keyboards for the bot
type KeyboardHandler struct {
	mainMonthKeyboard tgbotapi.InlineKeyboardMarkup
	allMonthsKeyboard tgbotapi.InlineKeyboardMarkup
}

// NewKeyboardHandler creates a new KeyboardHandler instance with cached keyboards
func NewKeyboardHandler() *KeyboardHandler {
	k := &KeyboardHandler{}

	// Создаём статическую клавиатуру для всех месяцев
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(months); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3 && i+j < len(months); j++ {
			month := months[i+j]
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(month, "month_"+strings.ToLower(month)))
		}
		rows = append(rows, row)
	}
	// Добавляем кнопку "Back"
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Back", "back_to_main")))
	k.allMonthsKeyboard = tgbotapi.NewInlineKeyboardMarkup(rows...)

	// Инициализируем mainMonthKeyboard
	k.updateMainMonthKeyboard()

	// Запускаем периодическое обновление mainMonthKeyboard 1-го числа каждого месяца
	go func() {
		for {
			// Вычисляем время до следующего 1-го числа
			now := time.Now()
			// Получаем начало следующего месяца
			nextMonth := now.AddDate(0, 1, 0)
			// Устанавливаем 1-е число следующего месяца, 00:00:00
			firstOfNextMonth := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location())
			// Вычисляем время ожидания
			durationUntilFirst := firstOfNextMonth.Sub(now)

			// Ждём до 1-го числа
			time.Sleep(durationUntilFirst)

			// Обновляем клавиатуру
			k.updateMainMonthKeyboard()
		}
	}()

	return k
}

// updateMainMonthKeyboard updates the main month keyboard with the current, previous, and next months
func (k *KeyboardHandler) updateMainMonthKeyboard() {
	// Main Month Keyboard: текущий, предыдущий и следующий месяц
	currentMonth := time.Now().Month()
	prevMonth := currentMonth - 1
	if prevMonth < 1 {
		prevMonth = 12
	}
	nextMonth := currentMonth + 1
	if nextMonth > 12 {
		nextMonth = 1
	}

	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(months[prevMonth-1], "month_"+strings.ToLower(months[prevMonth-1])),
		tgbotapi.NewInlineKeyboardButtonData(months[currentMonth-1], "month_"+strings.ToLower(months[currentMonth-1])),
		tgbotapi.NewInlineKeyboardButtonData(months[nextMonth-1], "month_"+strings.ToLower(months[nextMonth-1])),
		tgbotapi.NewInlineKeyboardButtonData("...", "show_all_months"),
	}

	k.mainMonthKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)
}

// GetMainMonthKeyboard returns the cached main month keyboard
func (k *KeyboardHandler) GetMainMonthKeyboard() tgbotapi.InlineKeyboardMarkup {
	return k.mainMonthKeyboard
}

// GetAllMonthsKeyboard returns the cached all months keyboard
func (k *KeyboardHandler) GetAllMonthsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return k.allMonthsKeyboard
}

// Stop is a no-op since periodic updates are managed with a simple sleep loop
func (k *KeyboardHandler) Stop() {
	// Ничего не делаем, так как бесконечный цикл завершится при остановке бота
}
