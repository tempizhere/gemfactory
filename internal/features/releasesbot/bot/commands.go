package bot

import (
    "fmt"
    "strings"

    "gemfactory/internal/debounce"
    "gemfactory/internal/features/releasesbot/artistlist"
    "gemfactory/internal/features/releasesbot/cache"
    "gemfactory/internal/features/releasesbot/release"
    "gemfactory/internal/features/releasesbot/releasefmt"
    "gemfactory/pkg/config"
    "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "go.uber.org/zap"
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
        {Command: "/start", Description: "Start the bot"},
        {Command: "/help", Description: "Show help message"},
        {Command: "/month", Description: "Get releases for a specific month"},
        {Command: "/whitelists", Description: "Show whitelists"},
        {Command: "/clearcache", Description: "Clear cache (admin only)"},
        {Command: "/add_artist", Description: "Add an artist to whitelist (admin only)"},
        {Command: "/remove_artist", Description: "Remove an artist from whitelist (admin only)"},
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
        h.handleStart(msg)
    case "help":
        h.handleHelp(msg)
    case "month":
        h.handleMonth(msg, args)
    case "whitelists":
        h.handleWhitelists(msg)
    case "clearcache":
        if !isAdmin {
            h.sendMessage(msg.Chat.ID, "This command is available only to admins.")
            return
        }
        h.handleClearCache(msg)
    case "add_artist":
        if !isAdmin {
            h.sendMessage(msg.Chat.ID, "This command is available only to admins.")
            return
        }
        h.handleAddArtist(msg, args)
    case "remove_artist":
        if !isAdmin {
            h.sendMessage(msg.Chat.ID, "This command is available only to admins.")
            return
        }
        h.handleRemoveArtist(msg, args)
    default:
        h.sendMessage(msg.Chat.ID, "Unknown command. Use /help to see available commands.")
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

// handleStart processes the /start command
func (h *CommandHandlers) handleStart(msg *tgbotapi.Message) {
    text := "Добро пожаловать! Выберите месяц:"
    reply := tgbotapi.NewMessage(msg.Chat.ID, text)
    reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
    h.sendMessageWithMarkup(msg.Chat.ID, text, reply.ReplyMarkup)
}

// handleHelp processes the /help command
func (h *CommandHandlers) handleHelp(msg *tgbotapi.Message) {
    text := "Доступные команды:\n" +
        "/start - Начать работу с ботом\n" +
        "/help - Показать это сообщение\n" +
        "/month [месяц] - Получить релизы за указанный месяц\n" +
        "/whitelists - Показать списки артистов\n" +
        "/clearcache - Очистить кэш (только для админа)\n" +
        "/add_artist <female|male> <имя> - Добавить артиста (только для админа)\n" +
        "/remove_artist <имя> - Удалить артиста (только для админа)"
    reply := tgbotapi.NewMessage(msg.Chat.ID, text)
    reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
    h.sendMessageWithMarkup(msg.Chat.ID, text, reply.ReplyMarkup)
}

// handleMonth processes the /month command
func (h *CommandHandlers) handleMonth(msg *tgbotapi.Message, args []string) {
    if len(args) == 0 {
        text := "Пожалуйста, выберите месяц:"
        reply := tgbotapi.NewMessage(msg.Chat.ID, text)
        reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
        h.sendMessageWithMarkup(msg.Chat.ID, text, reply.ReplyMarkup)
        return
    }

    month := strings.ToLower(args[0])
    validMonth := false
    for _, m := range release.Months {
        if month == m {
            validMonth = true
            break
        }
    }
    if !validMonth {
        h.sendMessage(msg.Chat.ID, "Неверный месяц. Используйте /month [january, february, ...]")
        return
    }

    whitelist := h.al.GetUnitedWhitelist()
    releases, err := cache.GetReleasesForMonths([]string{month}, whitelist, false, false, whitelist, h.config, h.logger)
    if err != nil {
        h.sendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при получении релизов: %v", err))
        return
    }

    if len(releases) == 0 {
        h.sendMessage(msg.Chat.ID, "Релизы не найдены.")
        return
    }

    var response strings.Builder
    for _, rel := range releases {
        formatted := releasefmt.FormatReleaseForTelegram(rel, h.logger)
        response.WriteString(formatted + "\n")
    }

    reply := tgbotapi.NewMessage(msg.Chat.ID, response.String())
    reply.ParseMode = "HTML"
    reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
    reply.DisableWebPagePreview = true // Отключаем превью ссылок
    h.sendMessageWithMarkup(msg.Chat.ID, response.String(), reply.ReplyMarkup)
}

// handleWhitelists processes the /whitelists command
func (h *CommandHandlers) handleWhitelists(msg *tgbotapi.Message) {
    female := h.al.GetFemaleWhitelist()
    male := h.al.GetMaleWhitelist()

    var response strings.Builder
    response.WriteString("<b>Женские артисты:</b>\n")
    for artist := range female {
        response.WriteString(artist + "\n")
    }
    response.WriteString("\n<b>Мужские артисты:</b>\n")
    for artist := range male {
        response.WriteString(artist + "\n")
    }

    reply := tgbotapi.NewMessage(msg.Chat.ID, response.String())
    reply.ParseMode = "HTML"
    reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
    h.sendMessageWithMarkup(msg.Chat.ID, response.String(), reply.ReplyMarkup)
}

// handleClearCache processes the /clearcache command
func (h *CommandHandlers) handleClearCache(msg *tgbotapi.Message) {
    cache.ClearCache()
    go cache.InitializeCache(h.config, h.logger, h.al)
    h.sendMessage(msg.Chat.ID, "Кэш очищен, обновление запущено.")
}

// handleAddArtist processes the /add_artist command
func (h *CommandHandlers) handleAddArtist(msg *tgbotapi.Message, args []string) {
    if len(args) < 2 {
        h.sendMessage(msg.Chat.ID, "Использование: /add_artist <female|male> <имя>")
        return
    }

    gender := strings.ToLower(args[0])
    isFemale := gender == "female"
    if gender != "female" && gender != "male" {
        h.sendMessage(msg.Chat.ID, "Первый аргумент должен быть 'female' или 'male'")
        return
    }

    artist := strings.Join(args[1:], " ")
    if err := h.al.AddArtist(artist, isFemale); err != nil {
        h.sendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при добавлении артиста: %v", err))
        return
    }

    h.sendMessage(msg.Chat.ID, fmt.Sprintf("Артист %s добавлен в %s whitelist", artist, gender))
}

// handleRemoveArtist processes the /remove_artist command
func (h *CommandHandlers) handleRemoveArtist(msg *tgbotapi.Message, args []string) {
    if len(args) < 1 {
        h.sendMessage(msg.Chat.ID, "Использование: /remove_artist <имя>")
        return
    }

    artist := strings.Join(args, " ")
    if err := h.al.RemoveArtist(artist); err != nil {
        h.sendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при удалении артиста: %v", err))
        return
    }

    h.sendMessage(msg.Chat.ID, fmt.Sprintf("Артист %s удалён из whitelist", artist))
}

// sendMessage sends a simple text message
func (h *CommandHandlers) sendMessage(chatID int64, text string) {
    msg := tgbotapi.NewMessage(chatID, text)
    if _, err := h.api.Send(msg); err != nil {
        h.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.String("text", text), zap.Error(err))
    }
}

// sendMessageWithMarkup sends a message with a reply markup
func (h *CommandHandlers) sendMessageWithMarkup(chatID int64, text string, markup interface{}) {
    msg := tgbotapi.NewMessage(chatID, text)
    msg.ReplyMarkup = markup
    msg.ParseMode = "HTML"
    msg.DisableWebPagePreview = true // Отключаем превью ссылок
    if _, err := h.api.Send(msg); err != nil {
        h.logger.Error("Failed to send message with markup", zap.Int64("chat_id", chatID), zap.String("text", text), zap.Error(err))
    }
}