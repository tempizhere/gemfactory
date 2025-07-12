package middleware

// RateLimiterInterface определяет интерфейс для ограничителя запросов
type RateLimiterInterface interface {
	// AllowRequest проверяет, можно ли обработать запрос
	AllowRequest(userID int64) bool

	// Cleanup очищает устаревшие записи
	Cleanup()
}
