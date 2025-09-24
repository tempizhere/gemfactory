// Package app содержит маршрутизацию команд.
package app

import (
	"gemfactory/internal/config"
	"gemfactory/internal/external/telegram"
	"gemfactory/internal/handlers"
	"gemfactory/internal/middleware"
	"gemfactory/internal/service"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Router обрабатывает маршрутизацию команд
type Router struct {
	handlers   *handlers.Handlers
	middleware *middleware.Middleware
	logger     *zap.Logger
}

// NewRouter создает новый роутер
func NewRouter(services *service.Services, config *config.Config, logger *zap.Logger) *Router {
	return &Router{
		handlers:   handlers.RegisterRoutes(services, config, logger),
		middleware: middleware.New(config, logger),
		logger:     logger,
	}
}

// NewRouterWithBotAPI создает новый роутер с BotAPI
func NewRouterWithBotAPI(services *service.Services, config *config.Config, logger *zap.Logger, botAPI telegram.BotAPI) *Router {
	return &Router{
		handlers:   handlers.RegisterRoutesWithBotAPI(services, config, logger, botAPI),
		middleware: middleware.New(config, logger),
		logger:     logger,
	}
}

// HandleUpdate обрабатывает обновление от Telegram
func (r *Router) HandleUpdate(update tgbotapi.Update) {
	// Применяем все middleware
	r.middleware.ProcessWithMiddleware(update, func(update tgbotapi.Update) {
		// Обработка сообщений
		if update.Message != nil {
			r.handleMessage(update.Message)
		}

		// Обработка callback query
		if update.CallbackQuery != nil {
			r.handleCallbackQuery(update.CallbackQuery)
		}
	})
}

// handleMessage обрабатывает текстовые сообщения
func (r *Router) handleMessage(message *tgbotapi.Message) {
	if !message.IsCommand() {
		return
	}

	command := strings.ToLower(message.Command())

	switch command {
	case "start":
		r.handlers.Start(message)
	case "help":
		r.handlers.Help(message)
	case "month":
		r.handlers.Month(message)
	case "search":
		r.handlers.Search(message)
	case "artists":
		r.handlers.Artists(message)
	case "metrics":
		r.handlers.Metrics(message)
	case "homework":
		r.handlers.Homework(message)
	case "playlist":
		r.handlers.Playlist(message)
	case "admin":
		r.handlers.Admin(message)
	case "add_artist":
		r.handlers.AddArtist(message)
	case "remove_artist":
		r.handlers.RemoveArtist(message)
	case "clearcache":
		r.handlers.ClearCache(message)
	case "clearwhitelists":
		r.handlers.ClearWhitelists(message)
	case "export":
		r.handlers.Export(message)
	case "config":
		r.handlers.Config(message)
	case "config_list":
		r.handlers.ConfigList(message)
	case "config_reset":
		r.handlers.ConfigReset(message)
	case "tasks_list":
		r.handlers.TasksList(message)
	case "reload_playlist":
		r.handlers.ReloadPlaylist(message)
	case "parse_releases":
		r.handlers.ParseReleases(message)
	default:
		r.handlers.Unknown(message)
	}
}

// handleCallbackQuery обрабатывает callback query
func (r *Router) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	r.handlers.CallbackQuery(query)
}

// RegisterBotCommands регистрирует команды бота
func (r *Router) RegisterBotCommands() []tgbotapi.BotCommand {
	return r.handlers.RegisterBotCommands()
}
