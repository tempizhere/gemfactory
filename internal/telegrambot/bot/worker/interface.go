// Package worker содержит интерфейсы для пула воркеров Telegram-бота.
package worker

import (
	"time"
)

// PoolInterface определяет интерфейс для пула воркеров
type PoolInterface interface {
	// Start запускает пул воркеров
	Start()

	// Stop останавливает пул воркеров
	Stop()

	// Submit добавляет задачу в очередь
	Submit(job Job) error

	// GetMetrics возвращает текущие метрики
	GetMetrics() Metrics

	// GetProcessedJobs возвращает количество обработанных задач
	GetProcessedJobs() int64

	// GetFailedJobs возвращает количество неудачных задач
	GetFailedJobs() int64

	// GetProcessingTime возвращает общее время обработки
	GetProcessingTime() time.Duration

	// GetQueueSize возвращает текущий размер очереди
	GetQueueSize() int
}
