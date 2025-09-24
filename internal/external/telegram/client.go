// Package telegram содержит интеграцию с Telegram Bot API.
package telegram

import (
	"context"
	"fmt"
	"gemfactory/internal/config"
	"gemfactory/internal/service"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// RouterInterface определяет интерфейс для роутера
type RouterInterface interface {
	HandleUpdate(update tgbotapi.Update)
	RegisterBotCommands() []tgbotapi.BotCommand
}

// Client представляет клиент Telegram Bot API
type Client struct {
	bot      *tgbotapi.BotAPI
	botAPI   BotAPI
	router   RouterInterface
	logger   *zap.Logger
	services *service.Services
	config   *config.Config
}

// NewClient создает новый клиент Telegram
func NewClient(botToken string, config *config.Config, logger *zap.Logger) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	bot.Debug = false
	logger.Info("Telegram bot created", zap.String("username", bot.Self.UserName))

	// Создаем BotAPI wrapper
	botAPI := NewTelegramBotAPI(bot, logger)

	return &Client{
		bot:    bot,
		botAPI: botAPI,
		logger: logger,
		config: config,
	}, nil
}

// Start запускает обработку обновлений
func (c *Client) Start(ctx context.Context, services *service.Services, router RouterInterface) error {
	c.services = services
	c.router = router

	// Инициализация бота
	c.logger.Info("Bot started", zap.String("username", c.bot.Self.UserName))

	// Удаляем webhook если есть
	_, err := c.bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	if err != nil {
		c.logger.Error("Failed to delete webhook", zap.Error(err))
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	// Настраиваем команды бота
	commands := c.router.RegisterBotCommands()
	_, err = c.bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		c.logger.Error("Failed to set bot commands", zap.Error(err))
		return fmt.Errorf("failed to set bot commands: %w", err)
	}

	// Настраиваем long polling
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}

	c.logger.Info("Starting to fetch updates")
	updatesChan := c.bot.GetUpdatesChan(u)
	if updatesChan == nil {
		return fmt.Errorf("failed to create updates channel")
	}

	reconnectDelay := 10 * time.Second // Задержка между попытками реконнекта

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Update loop cancelled by context")
			return ctx.Err()
		case update, ok := <-updatesChan:
			if !ok {
				c.logger.Warn("Update channel closed, will try to reconnect after delay")
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(reconnectDelay):
					return fmt.Errorf("update channel closed, reconnecting")
				}
			}

			// Обработка обновления
			c.processUpdate(update)
		}
	}
}

// processUpdate обрабатывает одно обновление
func (c *Client) processUpdate(update tgbotapi.Update) {
	// Улучшенное логирование с helper функциями
	c.logger.Debug("Processing update",
		zap.Int("update_id", update.UpdateID),
		zap.Int64("user_id", getUserID(update)),
		zap.String("command", extractCommand(update)),
		zap.String("update_type", getUpdateType(update)),
	)

	if update.Message != nil {
		c.logger.Debug("Received message",
			zap.String("text", update.Message.Text),
			zap.Int64("chat_id", update.Message.Chat.ID),
			zap.String("user", getUserIdentifier(update.Message.From)),
			zap.Int("update_id", update.UpdateID))
	} else if update.CallbackQuery != nil {
		month := extractMonth(update.CallbackQuery.Data)
		c.logger.Info("Received callback",
			zap.String("data", update.CallbackQuery.Data),
			zap.String("month", month),
			zap.Int64("chat_id", update.CallbackQuery.Message.Chat.ID),
			zap.String("user", getUserIdentifier(update.CallbackQuery.From)))
		c.logger.Debug("Callback details",
			zap.Int("update_id", update.UpdateID))
	}

	if update.Message == nil && update.CallbackQuery == nil {
		return
	}

	// Пропускаем вложения файлов (не обрабатываем)
	if update.Message != nil && update.Message.Document != nil {
		return
	}

	// Обрабатываем только команды
	if update.Message != nil && !update.Message.IsCommand() {
		return
	}

	// Обрабатываем обновление через роутер
	c.router.HandleUpdate(update)
}

// handleUpdate обрабатывает обновление (для совместимости)
func (c *Client) handleUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("Panic in handleUpdate", zap.Any("panic", r))
		}
	}()

	c.processUpdate(update)
}

// SendMessage отправляет сообщение
func (c *Client) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (c *Client) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard

	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}

// AnswerCallbackQuery отвечает на callback query
func (c *Client) AnswerCallbackQuery(callbackID string, text string) error {
	callback := tgbotapi.NewCallback(callbackID, text)
	_, err := c.bot.Request(callback)
	if err != nil {
		return fmt.Errorf("failed to answer callback query: %w", err)
	}

	return nil
}

// EditMessage редактирует сообщение
func (c *Client) EditMessage(chatID int64, messageID int, text string) error {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeMarkdown

	_, err := c.bot.Send(edit)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

// EditMessageWithKeyboard редактирует сообщение с клавиатурой
func (c *Client) EditMessageWithKeyboard(chatID int64, messageID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeMarkdown
	edit.ReplyMarkup = &keyboard

	_, err := c.bot.Send(edit)
	if err != nil {
		return fmt.Errorf("failed to edit message with keyboard: %w", err)
	}

	return nil
}

// DeleteMessage удаляет сообщение
func (c *Client) DeleteMessage(chatID int64, messageID int) error {
	delete := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := c.bot.Send(delete)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// GetBotInfo возвращает информацию о боте
func (c *Client) GetBotInfo() *tgbotapi.User {
	return &c.bot.Self
}

// GetBotAPI возвращает BotAPI интерфейс
func (c *Client) GetBotAPI() BotAPI {
	return c.botAPI
}

// Helper функции для логирования

// getUserID извлекает ID пользователя из обновления
func getUserID(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.From.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID
	}
	return 0
}

// extractCommand извлекает команду из обновления
func extractCommand(update tgbotapi.Update) string {
	if update.Message != nil && update.Message.IsCommand() {
		return update.Message.Command()
	}
	if update.CallbackQuery != nil {
		return "callback"
	}
	return ""
}

// getUpdateType определяет тип обновления
func getUpdateType(update tgbotapi.Update) string {
	if update.Message != nil {
		if update.Message.IsCommand() {
			return "command"
		}
		return "message"
	}
	if update.CallbackQuery != nil {
		return "callback"
	}
	return "unknown"
}

// getUserIdentifier возвращает идентификатор пользователя
func getUserIdentifier(user *tgbotapi.User) string {
	if user == nil {
		return "unknown"
	}

	if user.UserName != "" {
		return "@" + user.UserName
	}

	if user.FirstName != "" {
		if user.LastName != "" {
			return user.FirstName + " " + user.LastName
		}
		return user.FirstName
	}

	return fmt.Sprintf("user_%d", user.ID)
}

// extractMonth извлекает месяц из callback data
func extractMonth(data string) string {
	if strings.HasPrefix(data, "month_") {
		return strings.TrimPrefix(data, "month_")
	}
	return "unknown"
}
