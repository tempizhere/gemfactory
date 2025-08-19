// Package playlist содержит компоненты для уведомления админа.
package playlist

import (
	"fmt"
	"gemfactory/internal/gateway/telegram/botapi"

	"go.uber.org/zap"
)

// AdminNotifier реализует BotAPIInterface для отправки сообщений админу
type AdminNotifier struct {
	botAPI        botapi.BotAPI
	adminUsername string
	adminChatID   int64 // Кэш chat ID админа
	logger        *zap.Logger
}

// NewAdminNotifier создает новый уведомитель админа
func NewAdminNotifier(botAPI botapi.BotAPI, adminUsername string, logger *zap.Logger) *AdminNotifier {
	return &AdminNotifier{
		botAPI:        botAPI,
		adminUsername: adminUsername,
		logger:        logger,
	}
}

// SendMessageToAdmin отправляет сообщение администратору
func (a *AdminNotifier) SendMessageToAdmin(message string) error {
	if a.botAPI == nil {
		return fmt.Errorf("bot API is not available")
	}

	if a.adminUsername == "" {
		return fmt.Errorf("admin username is not configured")
	}

	// Для простоты пока логируем сообщение
	// В реальной реализации нужно было бы найти chat ID по username
	// или использовать другой способ отправки админу
	a.logger.Warn("Admin notification (not implemented - would send to admin)",
		zap.String("admin_username", a.adminUsername),
		zap.String("message", message))

	// TODO: Реализовать реальную отправку сообщения админу
	// Можно использовать хранение chat ID админа в базе данных
	// или отправлять через webhook/email

	return nil
}
