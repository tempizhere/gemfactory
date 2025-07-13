package metrics

import "time"

// Interface определяет интерфейс для системы метрик
type Interface interface {
	// RecordUserCommand записывает выполнение пользовательской команды
	RecordUserCommand(command string, userID int64)

	// UpdateArtistMetrics обновляет метрики артистов
	UpdateArtistMetrics(femaleCount, maleCount int)

	// UpdateReleaseMetrics обновляет метрики релизов
	UpdateReleaseMetrics(cachedCount int)

	// RecordCacheHit записывает попадание в кэш
	RecordCacheHit()

	// RecordCacheMiss записывает промах кэша
	RecordCacheMiss()

	// RecordResponseTime записывает время ответа
	RecordResponseTime(duration time.Duration)

	// RecordError записывает ошибку
	RecordError()

	// SetCacheUpdateStatus устанавливает статус обновления кэша
	SetCacheUpdateStatus(updating bool)

	// SetNextCacheUpdate устанавливает время следующего обновления кэша
	SetNextCacheUpdate(nextUpdate time.Time)

	// GetStats возвращает все метрики в виде map
	GetStats() map[string]interface{}
}
