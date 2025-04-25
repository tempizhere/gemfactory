package bot

import (
	"fmt"
	"sort"
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
	case "clearwhitelists":
		if !isAdmin {
			h.sendMessage(msg.Chat.ID, "This command is available only to admins.")
			return
		}
		h.handleClearWhitelists(msg)
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
		"/month [месяц] -gg - Получить релизы только для женских групп\n" +
		"/month [месяц] -mg - Получить релизы только для мужских групп\n" +
		"/whitelists - Показать списки артистов"
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
	response.WriteString("<b>Женские артисты:</b> ")
	femaleArtists := make([]string, 0, len(female))
	for artist := range female {
		femaleArtists = append(femaleArtists, artist)
	}
	sort.Strings(femaleArtists) // Сортируем для упорядоченного вывода
	if len(femaleArtists) == 0 {
		response.WriteString("пусто")
	} else {
		response.WriteString(strings.Join(femaleArtists, ", "))
	}

	response.WriteString("\n<b>Мужские артисты:</b> ")
	maleArtists := make([]string, 0, len(male))
	for artist := range male {
		maleArtists = append(maleArtists, artist)
	}
	sort.Strings(maleArtists) // Сортируем для упорядоченного вывода
	if len(maleArtists) == 0 {
		response.WriteString("пусто")
	} else {
		response.WriteString(strings.Join(maleArtists, ", "))
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
		h.sendMessage(msg.Chat.ID, "Использование: /add_artist <female|male> <artist1,artist2,...>")
		return
	}

	gender := strings.ToLower(args[0])
	isFemale := gender == "female"
	if gender != "female" && gender != "male" {
		h.sendMessage(msg.Chat.ID, "Первый аргумент должен быть 'female' или 'male'. Пример: /add_artist female ITZY,aespa,IVE")
		return
	}

	// Объединяем аргументы, начиная со второго, и парсим список артистов
	artistsInput := strings.Join(args[1:], " ")
	artists := parseArtists(artistsInput)
	if len(artists) == 0 {
		h.sendMessage(msg.Chat.ID, "Не указаны артисты для добавления")
		return
	}

	addedCount, err := h.al.AddArtists(artists, isFemale)
	if err != nil {
		h.sendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при добавлении артистов: %v", err))
		return
	}

	if addedCount == 0 {
		h.sendMessage(msg.Chat.ID, "Ни один артист не добавлен, так как все указанные артисты уже в whitelist")
		return
	}

	artistWord := "артист"
	if addedCount > 1 && addedCount < 5 {
		artistWord = "артиста"
	} else if addedCount >= 5 {
		artistWord = "артистов"
	}
	h.sendMessage(msg.Chat.ID, fmt.Sprintf("Добавлено %d %s в %s whitelist", addedCount, artistWord, gender))

	// Запускаем обновление кэша асинхронно
	go cache.InitializeCache(h.config, h.logger, h.al)
}

// handleRemoveArtist processes the /remove_artist command
func (h *CommandHandlers) handleRemoveArtist(msg *tgbotapi.Message, args []string) {
	if len(args) < 1 {
		h.sendMessage(msg.Chat.ID, "Использование: /remove_artist <artist1,artist2,...>")
		return
	}

	// Объединяем аргументы и парсим список артистов
	artistsInput := strings.Join(args, " ")
	artists := parseArtists(artistsInput)
	if len(artists) == 0 {
		h.sendMessage(msg.Chat.ID, "Не указаны артисты для удаления")
		return
	}

	removedCount, err := h.al.RemoveArtists(artists)
	if err != nil {
		h.sendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при удалении артистов: %v", err))
		return
	}

	if removedCount == 0 {
		h.sendMessage(msg.Chat.ID, "Ни один артист не удалён, так как указанные артисты отсутствуют в whitelist")
		return
	}

	artistWord := "артист"
	if removedCount > 1 && removedCount < 5 {
		artistWord = "артиста"
	} else if removedCount >= 5 {
		artistWord = "артистов"
	}
	h.sendMessage(msg.Chat.ID, fmt.Sprintf("Удалено %d %s из whitelist", removedCount, artistWord))

	// Запускаем обновление кэша асинхронно
	go cache.InitializeCache(h.config, h.logger, h.al)
}

// handleClearWhitelists processes the /clearwhitelists command
func (h *CommandHandlers) handleClearWhitelists(msg *tgbotapi.Message) {
	if err := h.al.ClearWhitelists(); err != nil {
		h.sendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при очистке вайтлистов: %v", err))
		return
	}
	h.sendMessage(msg.Chat.ID, "Вайтлисты очищены")
}

// parseArtists parses a comma-separated list of artists, handling spaces and special characters
func parseArtists(input string) []string {
	// Разделяем по запятым, учитывая пробелы
	rawArtists := strings.Split(input, ",")
	var artists []string
	for _, artist := range rawArtists {
		// Очищаем от пробелов
		cleaned := strings.TrimSpace(artist)
		if cleaned != "" {
			artists = append(artists, cleaned)
		}
	}
	return artists
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
