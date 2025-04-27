package bot

import (
	"fmt"
	"gemfactory/internal/debounce"
	"gemfactory/internal/features/releasesbot/artistlist"
	"gemfactory/pkg/config"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"strings"
)

// CommandHandlers handles Telegram commands
type CommandHandlers struct {
	api       *tgbotapi.BotAPI
	logger    *zap.Logger
	config    *config.Config
	al        *artistlist.ArtistList
	keyboard  *KeyboardManager
	debouncer *debounce.Debouncer
}

// NewCommandHandlers creates a new CommandHandlers instance
func NewCommandHandlers(api *tgbotapi.BotAPI, logger *zap.Logger, debouncer *debounce.Debouncer, config *config.Config, al *artistlist.ArtistList) *CommandHandlers {
	keyboard := NewKeyboardManager(api, logger, al, config)
	return &CommandHandlers{
		api:       api,
		logger:    logger,
		config:    config,
		al:        al,
		keyboard:  keyboard,
		debouncer: debouncer,
	}
}

// SetBotCommands sets the bot's command menu
func (h *CommandHandlers) SetBotCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "/help", Description: "Show help message"},
		{Command: "/month", Description: "Get releases for a specific month"},
		{Command: "/whitelists", Description: "Show whitelists"},
	}

	_, err := h.api.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		return err
	}
	h.logger.Info("Bot commands set successfully")
	return nil
}

// HandleCommand processes incoming commands
func (h *CommandHandlers) HandleCommand(update tgbotapi.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	command := strings.ToLower(msg.Command())
	args := strings.Fields(msg.Text)[1:]

	isAdmin := msg.From.UserName == h.config.AdminUsername

	switch command {
	case "start":
		handleStart(h, msg)
	case "help":
		handleHelp(h, msg)
	case "month":
		handleMonth(h, msg, args)
	case "whitelists":
		handleWhitelists(h, msg)
	case "clearcache":
		if !isAdmin {
			sendMessage(h, msg.Chat.ID, "This command is available only to admins.")
			return
		}
		handleClearCache(h, msg)
	case "add_artist":
		if !isAdmin {
			sendMessage(h, msg.Chat.ID, "This command is available only to admins.")
			return
		}
		handleAddArtist(h, msg, args)
	case "remove_artist":
		if !isAdmin {
			sendMessage(h, msg.Chat.ID, "This command is available only to admins.")
			return
		}
		handleRemoveArtist(h, msg, args)
	case "clearwhitelists":
		if !isAdmin {
			sendMessage(h, msg.Chat.ID, "This command is available only to admins.")
			return
		}
		handleClearWhitelists(h, msg)
	default:
		sendMessage(h, msg.Chat.ID, "Unknown command. Use /help to see available commands.")
	}
}

// HandleCallbackQuery processes callback queries from inline keyboards
func (h *CommandHandlers) HandleCallbackQuery(update tgbotapi.Update) {
	callback := update.CallbackQuery
	if callback == nil {
		return
	}

	// Ответ на callback query, чтобы убрать "часики"
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	if _, err := h.api.Request(callbackConfig); err != nil {
		h.logger.Error("Failed to answer callback query", zap.Error(err))
	}

	key := fmt.Sprintf("%d-%s", callback.From.ID, callback.Data)
	if !h.debouncer.CanProcessRequest(key) {
		h.logger.Debug("Double-click prevented", zap.String("user", callback.From.UserName), zap.String("data", callback.Data))
		return
	}

	h.keyboard.HandleCallback(callback)
}
