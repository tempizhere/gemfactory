package health

import "context"

// ServerInterface определяет интерфейс для health check сервера
type ServerInterface interface {
	// Start запускает health check сервер
	Start() error

	// Stop останавливает health check сервер
	Stop(ctx context.Context) error
}
