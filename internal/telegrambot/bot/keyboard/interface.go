package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// KeyboardManagerInterface определяет интерфейс для управления клавиатурами
type KeyboardManagerInterface interface {
	// GetMainKeyboard возвращает основную клавиатуру с месяцами
	GetMainKeyboard() tgbotapi.InlineKeyboardMarkup

	// GetAllMonthsKeyboard возвращает клавиатуру со всеми месяцами
	GetAllMonthsKeyboard() tgbotapi.InlineKeyboardMarkup

	// HandleCallbackQuery обрабатывает callback запросы от inline клавиатур
	HandleCallbackQuery(callback *tgbotapi.CallbackQuery)

	// StartWorkerPool запускает worker pool для keyboard manager
	StartWorkerPool()

	// StopWorkerPool останавливает worker pool для keyboard manager
	StopWorkerPool()

	// Stop останавливает keyboard manager
	Stop()
}
