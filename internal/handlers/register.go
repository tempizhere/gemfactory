// Package handlers содержит регистрацию маршрутов.
package handlers

import (
	"gemfactory/internal/config"
	"gemfactory/internal/external/telegram"
	"gemfactory/internal/keyboard"
	"gemfactory/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// RegisterRoutes регистрирует все маршруты
func RegisterRoutes(services *service.Services, config *config.Config, logger *zap.Logger) *Handlers {
	// Создаем менеджер клавиатур
	keyboardManager := keyboard.NewKeyboardManager(services, config, logger)

	// Создаем обработчики с клавиатурой (BotAPI будет установлен позже)
	handlers := New(services, config, keyboardManager, logger)

	return handlers
}

// RegisterRoutesWithBotAPI регистрирует все маршруты с BotAPI
func RegisterRoutesWithBotAPI(services *service.Services, config *config.Config, logger *zap.Logger, botAPI telegram.BotAPI) *Handlers {
	// Создаем менеджер клавиатур
	keyboardManager := keyboard.NewKeyboardManager(services, config, logger)

	// Устанавливаем BotAPI в keyboard manager
	keyboardManager.SetBotAPI(botAPI)

	// Создаем обработчики с клавиатурой и BotAPI
	handlers := New(services, config, keyboardManager, logger)
	handlers.botAPI = botAPI

	return handlers
}

// RegisterBotCommands регистрирует команды бота
func (h *Handlers) RegisterBotCommands() []tgbotapi.BotCommand {
	return []tgbotapi.BotCommand{
		{Command: "start", Description: "Начать работу с ботом"},
		{Command: "help", Description: "Показать справку"},
		{Command: "month", Description: "Получить релизы за месяц"},
		{Command: "search", Description: "Поиск релизов по артисту"},
		{Command: "artists", Description: "Показать списки артистов"},
		{Command: "metrics", Description: "Показать метрики системы"},
		{Command: "homework", Description: "Получить случайное домашнее задание"},
		{Command: "playlist", Description: "Информация о плейлисте"},
	}
}
