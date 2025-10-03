package telegram

// ConfigInterface определяет интерфейс для конфигурации
type ConfigInterface interface {
	GetBotToken() string
	GetAdminUsername() string
}
