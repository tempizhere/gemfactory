// Package types содержит основные типы данных для Telegram-бота.
package types

import (
	"errors"
	"fmt"
	"gemfactory/internal/bot/keyboard"
	"gemfactory/internal/bot/service"
	"gemfactory/internal/config"
	"gemfactory/internal/gateway/telegram/botapi"
	cachemodule "gemfactory/internal/infrastructure/cache"
	"gemfactory/internal/infrastructure/debounce"
	"gemfactory/internal/infrastructure/metrics"
	"gemfactory/internal/infrastructure/worker"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Стандартные ошибки бота
var (
	ErrCommandNotFound   = errors.New("command not found")
	ErrInvalidArguments  = errors.New("invalid arguments")
	ErrUnauthorized      = errors.New("unauthorized access")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrBotNotStarted     = errors.New("bot not started")
	ErrBotAlreadyStarted = errors.New("bot already started")
	ErrContextCancelled  = errors.New("context cancelled")
)

// Error codes для BotError
const (
	ErrCodeCommandNotFound   = "COMMAND_NOT_FOUND"
	ErrCodeInvalidArguments  = "INVALID_ARGUMENTS"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrCodeBotNotStarted     = "BOT_NOT_STARTED"
	ErrCodeBotAlreadyStarted = "BOT_ALREADY_STARTED"
	ErrCodeContextCancelled  = "CONTEXT_CANCELLED"
)

// BotError представляет ошибку бота с контекстом
type BotError struct {
	Code    string
	Message string
	Err     error
}

func (e *BotError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *BotError) Unwrap() error {
	return e.Err
}

// NewBotError создает новую ошибку бота
func NewBotError(code, message string, err error) *BotError {
	return &BotError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// CommandError представляет ошибку выполнения команды
type CommandError struct {
	Command string
	UserID  int64
	ChatID  int64
	Err     error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command %s failed for user %d in chat %d: %v",
		e.Command, e.UserID, e.ChatID, e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// IsCommandError проверяет, является ли ошибка CommandError
func IsCommandError(err error) bool {
	_, ok := err.(*CommandError)
	return ok
}

// NewCommandError создает новую ошибку команды
func NewCommandError(command string, userID, chatID int64, err error) *CommandError {
	return &CommandError{
		Command: command,
		UserID:  userID,
		ChatID:  chatID,
		Err:     err,
	}
}

// IsBotError проверяет, является ли ошибка BotError
func IsBotError(err error) bool {
	_, ok := err.(*BotError)
	return ok
}

// HandlerFunc defines a command handler function
type HandlerFunc func(ctx Context) error

// Middleware defines a middleware function
type Middleware func(ctx Context, next HandlerFunc) error

// Dependencies holds all bot dependencies
type Dependencies struct {
	BotAPI         botapi.BotAPI
	Logger         *zap.Logger
	Config         config.Interface
	ReleaseService service.ReleaseServiceInterface
	ArtistService  service.ArtistServiceInterface
	Keyboard       keyboard.ManagerInterface
	Debouncer      debounce.DebouncerInterface
	Cache          cachemodule.Cache
	WorkerPool     worker.PoolInterface

	Metrics metrics.Interface
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
		{Command: "/metrics", Description: "Показать метрики системы"},
		{Command: "/clearcache", Description: "Очистить кэш (только для админов)"},
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

// Interface определяет публичные методы Telegram-бота
// (минимальный контракт для использования в других пакетах)
type Interface interface {
	Start() error
	Stop() error
}
