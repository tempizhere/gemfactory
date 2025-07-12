package debounce

import (
	"sync"
	"time"
)

// Debouncer prevents double-clicks by rate-limiting requests
type Debouncer struct {
	lastRequest map[string]time.Time
	mu          sync.Mutex
}

// Убеждаемся, что Debouncer реализует DebouncerInterface
var _ DebouncerInterface = (*Debouncer)(nil)

const debounceTimeout = 5 * time.Second

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

	if time.Since(last) < debounceTimeout {
		return false
	}

	d.lastRequest[key] = time.Now()
	return true
}
