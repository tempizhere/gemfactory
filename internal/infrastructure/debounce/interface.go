package debounce

// DebouncerInterface определяет интерфейс для debouncer
type DebouncerInterface interface {
	// CanProcessRequest проверяет, можно ли обработать запрос
	CanProcessRequest(key string) bool
}
