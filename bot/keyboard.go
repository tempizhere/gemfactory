package bot

import (
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// KeyboardHandler manages Inline Keyboards for the bot
type KeyboardHandler struct {
	mainMonthKeyboard tgbotapi.InlineKeyboardMarkup
	allMonthsKeyboard tgbotapi.InlineKeyboardMarkup
	mu                sync.RWMutex
	stopChan          chan struct{}
	stopOnce          sync.Once
}

// NewKeyboardHandler creates a new KeyboardHandler instance with cached keyboards
func NewKeyboardHandler() *KeyboardHandler {
	k := &KeyboardHandler{
		stopChan: make(chan struct{}),
	}
	k.updateKeyboards() // Инициализируем клавиатуры при создании

	// Запускаем периодическое обновление клавиатур каждые 10 дней
	go func() {
		ticker := time.NewTicker(10 * 24 * time.Hour) // 10 дней
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				k.updateKeyboards()
			case <-k.stopChan:
				return
			}
		}
	}()

	return k
}

// updateKeyboards updates the cached keyboards
func (k *KeyboardHandler) updateKeyboards() {
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

	months := []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(months[prevMonth-1], "month_"+strings.ToLower(months[prevMonth-1])),
		tgbotapi.NewInlineKeyboardButtonData(months[currentMonth-1], "month_"+strings.ToLower(months[currentMonth-1])),
		tgbotapi.NewInlineKeyboardButtonData(months[nextMonth-1], "month_"+strings.ToLower(months[nextMonth-1])),
		tgbotapi.NewInlineKeyboardButtonData("...", "show_all_months"),
	}

	// Обновляем клавиатуры с использованием мьютекса
	k.mu.Lock()
	k.mainMonthKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)

	// All Months Keyboard: все 12 месяцев
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
	k.mu.Unlock()
}

// GetMainMonthKeyboard returns the cached main month keyboard
func (k *KeyboardHandler) GetMainMonthKeyboard() tgbotapi.InlineKeyboardMarkup {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.mainMonthKeyboard
}

// GetAllMonthsKeyboard returns the cached all months keyboard
func (k *KeyboardHandler) GetAllMonthsKeyboard() tgbotapi.InlineKeyboardMarkup {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.allMonthsKeyboard
}

// Stop stops the keyboard update ticker
func (k *KeyboardHandler) Stop() {
	k.stopOnce.Do(func() {
		close(k.stopChan)
	})
}
