// Package debounce реализует механизм дебаунса для предотвращения повторных запросов.
package debounce

import (
	"strings"
	"sync"
	"time"
)

// Debouncer prevents double-clicks by rate-limiting requests
type Debouncer struct {
	lastRequest map[string]time.Time
	mu          sync.Mutex
}

var _ DebouncerInterface = (*Debouncer)(nil)

const defaultDebounceTimeout = 5 * time.Second

// Команды с особыми таймаутами дебаунса
var commandDebounceTimeouts = map[string]time.Duration{
	"month": 5 * time.Second, // 5 секунд дебаунс для month
}

// NewDebouncer creates a new Debouncer instance
func NewDebouncer() *Debouncer {
	return &Debouncer{
		lastRequest: make(map[string]time.Time),
	}
}

// CanProcessRequest checks if a request can be processed based on the last request time
func (d *Debouncer) CanProcessRequest(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	last, exists := d.lastRequest[key]
	if !exists {
		d.lastRequest[key] = time.Now()
		return true
	}

	// Определяем таймаут для команды
	timeout := defaultDebounceTimeout
	command, _ := extractCommandFromKey(key)
	if customTimeout, exists := commandDebounceTimeouts[command]; exists {
		timeout = customTimeout
	}

	// Проверяем, прошло ли достаточно времени
	if time.Since(last) < timeout {
		return false
	}

	d.lastRequest[key] = time.Now()
	return true
}

// extractCommandFromKey извлекает команду из ключа дебаунса
func extractCommandFromKey(key string) (string, bool) {
	// Ключ имеет формат: "chatID:command"
	parts := strings.Split(key, ":")
	if len(parts) >= 2 {
		return parts[1], true
	}
	return "", false
}
