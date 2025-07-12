package bot

// BotInterface определяет интерфейс для основного бота
type BotInterface interface {
	// Start запускает бота
	Start() error

	// Stop останавливает бота
	Stop() error
}
