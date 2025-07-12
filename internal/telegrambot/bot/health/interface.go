package health

import "context"

// HealthServerInterface определяет интерфейс для health check сервера
type HealthServerInterface interface {
	// Start запускает health check сервер
	Start() error

	// Stop останавливает health check сервер
	Stop(ctx context.Context) error
}
