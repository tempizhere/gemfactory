package bot

// Interface определяет интерфейс для основного бота
type Interface interface {
	// Start запускает бота
	Start() error

	// Stop останавливает бота
	Stop() error
}
