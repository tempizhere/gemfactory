// Package keyboard содержит интерфейсы для управления клавиатурами Telegram-бота.
package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// ManagerInterface определяет интерфейс для менеджера клавиатур Telegram-бота.
type ManagerInterface interface {
	StartWorkerPool()
	StopWorkerPool()
	GetMainKeyboard() tgbotapi.InlineKeyboardMarkup
	GetAllMonthsKeyboard() tgbotapi.InlineKeyboardMarkup
	HandleCallbackQuery(callback *tgbotapi.CallbackQuery)
	Stop()
}
