// Package handlers содержит обработчики команд.
package handlers

import (
	"gemfactory/internal/external/telegram"
	"gemfactory/internal/keyboard"
	"gemfactory/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Handlers содержит все обработчики команд
type Handlers struct {
	services *service.Services
	logger   *zap.Logger
	keyboard keyboard.ManagerInterface
	botAPI   telegram.BotAPI
}

// New создает новый экземпляр обработчиков
func New(services *service.Services, keyboard keyboard.ManagerInterface, logger *zap.Logger) *Handlers {
	return &Handlers{
		services: services,
		logger:   logger,
		keyboard: keyboard,
	}
}

// isAdmin проверяет, является ли пользователь администратором
func (h *Handlers) isAdmin(user *tgbotapi.User) bool {
	// Получаем username администратора из конфигурации
	adminUsername, err := h.services.Config.GetConfigValue("ADMIN_USERNAME")
	if err != nil {
		h.logger.Warn("Failed to get admin username from config", zap.Error(err))
		return false
	}

	if adminUsername == "" {
		h.logger.Warn("ADMIN_USERNAME not configured")
		return false
	}

	// Проверяем username пользователя
	if user.UserName == "" {
		h.logger.Warn("User has no username", zap.Int64("user_id", user.ID))
		return false
	}

	return user.UserName == adminUsername
}
