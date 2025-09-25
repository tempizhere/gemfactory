// Package middleware содержит middleware для debounce.
package middleware

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Команды с особыми таймаутами дебаунса
var commandDebounceTimeouts = map[string]time.Duration{
	"month": 5 * time.Second, // 5 секунд дебаунс для month
}

// DebouncerInterface определяет интерфейс для debouncer
type DebouncerInterface interface {
	// CanProcessRequest проверяет, можно ли обработать запрос
	CanProcessRequest(key string) bool
	// CanProcessRequestWithTimeout проверяет, можно ли обработать запрос с кастомным таймаутом
	CanProcessRequestWithTimeout(key string, timeout time.Duration) bool
	// Cleanup очищает устаревшие записи
	Cleanup()
}

// Debouncer предотвращает двойные клики с таймаутом контекста
type Debouncer struct {
	requests map[string]time.Time
	mu       sync.RWMutex
	timeout  time.Duration
	logger   *zap.Logger
}

var _ DebouncerInterface = (*Debouncer)(nil)

// NewDebouncer создает новый debouncer
func NewDebouncer(timeout time.Duration, logger *zap.Logger) *Debouncer {
	return &Debouncer{
		requests: make(map[string]time.Time),
		timeout:  timeout,
		logger:   logger,
	}
}

// CanProcessRequest проверяет, можно ли обработать запрос
func (d *Debouncer) CanProcessRequest(key string) bool {
	return d.CanProcessRequestWithTimeout(key, d.timeout)
}

// CanProcessRequestWithTimeout проверяет, можно ли обработать запрос с кастомным таймаутом
func (d *Debouncer) CanProcessRequestWithTimeout(key string, timeout time.Duration) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	lastRequest, exists := d.requests[key]

	if !exists || now.Sub(lastRequest) > timeout {
		d.requests[key] = now
		return true
	}

	return false
}

// Cleanup очищает устаревшие записи
func (d *Debouncer) Cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for key, lastRequest := range d.requests {
		if now.Sub(lastRequest) > d.timeout {
			delete(d.requests, key)
		}
	}
}

// DebounceMiddleware предотвращает двойные клики с контекстным таймаутом
func DebounceMiddleware(debouncer DebouncerInterface, logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update)) {
	return func(update tgbotapi.Update, next func(tgbotapi.Update)) {
		if update.Message == nil {
			next(update)
			return
		}

		command := update.Message.Command()
		key := fmt.Sprintf("%d:%s", update.Message.Chat.ID, command)

		// Проверяем, есть ли кастомный таймаут для команды
		timeout, hasCustomTimeout := commandDebounceTimeouts[command]
		var canProcess bool

		if hasCustomTimeout {
			canProcess = debouncer.CanProcessRequestWithTimeout(key, timeout)
		} else {
			canProcess = debouncer.CanProcessRequest(key)
		}

		if !canProcess {
			user := getUserIdentifier(update.Message.From)
			logger.Info("Command debounced",
				zap.String("command", command),
				zap.Int64("chat_id", update.Message.Chat.ID),
				zap.String("user", user),
				zap.Int("update_id", update.UpdateID),
				zap.Duration("timeout", timeout))

			return
		}

		next(update)
	}
}

// DebounceMiddlewareWithError предотвращает двойные клики с обработкой ошибок
func DebounceMiddlewareWithError(debouncer DebouncerInterface, logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
	return func(update tgbotapi.Update, next func(tgbotapi.Update) error) error {
		if update.Message == nil {
			return next(update)
		}

		command := update.Message.Command()
		key := fmt.Sprintf("%d:%s", update.Message.Chat.ID, command)

		// Проверяем, есть ли кастомный таймаут для команды
		timeout, hasCustomTimeout := commandDebounceTimeouts[command]
		var canProcess bool

		if hasCustomTimeout {
			canProcess = debouncer.CanProcessRequestWithTimeout(key, timeout)
		} else {
			canProcess = debouncer.CanProcessRequest(key)
		}

		if !canProcess {
			user := getUserIdentifier(update.Message.From)
			logger.Info("Command debounced",
				zap.String("command", command),
				zap.Int64("chat_id", update.Message.Chat.ID),
				zap.String("user", user),
				zap.Int("update_id", update.UpdateID),
				zap.Duration("timeout", timeout))

			return nil
		}

		return next(update)
	}
}

// DebounceCallbackMiddleware предотвращает двойные клики на кнопки с контекстным таймаутом
func DebounceCallbackMiddleware(debouncer DebouncerInterface, logger *zap.Logger) func(update tgbotapi.Update, next func(tgbotapi.Update)) {
	return func(update tgbotapi.Update, next func(tgbotapi.Update)) {
		if update.CallbackQuery == nil {
			next(update)
			return
		}

		callbackData := update.CallbackQuery.Data
		if callbackData == "" {
			callbackData = "callback"
		}

		// Проверяем, нужно ли дебаунсить этот callback
		// Дебаунсим только кнопки месяцев (month_*)
		shouldDebounce := false
		var timeout time.Duration

		if strings.HasPrefix(callbackData, "month_") {
			shouldDebounce = true
			timeout = commandDebounceTimeouts["month"]
		}

		key := fmt.Sprintf("%d:%s", update.CallbackQuery.Message.Chat.ID, callbackData)
		var canProcess bool

		if shouldDebounce {
			canProcess = debouncer.CanProcessRequestWithTimeout(key, timeout)
		} else {
			// Для других callback'ов не применяем дебаунс
			canProcess = true
		}

		if !canProcess {
			user := getUserIdentifier(update.CallbackQuery.From)
			logger.Info("Callback debounced",
				zap.String("callback_data", callbackData),
				zap.Int64("chat_id", update.CallbackQuery.Message.Chat.ID),
				zap.String("user", user),
				zap.Int("update_id", update.UpdateID),
				zap.Duration("timeout", timeout))

			return
		}

		next(update)
	}
}
