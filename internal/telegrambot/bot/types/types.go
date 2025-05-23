package types

import (
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	"gemfactory/internal/telegrambot/bot/keyboard"
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/pkg/config"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// HandlerFunc defines a command handler function
type HandlerFunc func(ctx Context) error

// Middleware defines a middleware function
type Middleware func(ctx Context, next HandlerFunc) error

// Dependencies holds all bot dependencies
type Dependencies struct {
	BotAPI         botapi.BotAPI
	Logger         *zap.Logger
	Config         *config.Config
	ReleaseService service.ReleaseServiceInterface
	ArtistService  service.ArtistServiceInterface
	Keyboard       *keyboard.KeyboardManager
	Debouncer      *debounce.Debouncer
	Cache          cache.Cache
}

// Context holds the context for command handlers
type Context struct {
	Message         *tgbotapi.Message
	UpdateID        int
	Deps            *Dependencies
	HandlerExecuted bool // Tracks if handler has been executed
}

// SetBotCommands sets the bot's command menu
func (d *Dependencies) SetBotCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "/help", Description: "Показать справку"},
		{Command: "/month", Description: "Получить релизы за месяц"},
		{Command: "/whitelists", Description: "Показать списки артистов"},
	}
	if err := d.BotAPI.SetBotCommands(commands); err != nil {
		return err
	}
	d.Logger.Info("Bot commands set successfully")
	return nil
}

// GetUserIdentifier returns the username (if available) or name of the user
func GetUserIdentifier(user *tgbotapi.User) string {
	if user == nil {
		return "unknown"
	}
	if user.UserName != "" {
		return "@" + user.UserName
	}
	nameParts := []string{}
	if user.FirstName != "" {
		nameParts = append(nameParts, user.FirstName)
	}
	if user.LastName != "" {
		nameParts = append(nameParts, user.LastName)
	}
	if len(nameParts) > 0 {
		return strings.Join(nameParts, " ")
	}
	return "unknown"
}