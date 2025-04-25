package bot

import (
	"fmt"
	"html"
	"os"
	"strings"

	"gemfactory/models"
	"gemfactory/parser"
	"gemfactory/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// CommandHandlers handles Telegram commands and callback queries
type CommandHandlers struct {
	api       *tgbotapi.BotAPI
	logger    *zap.Logger
	keyboard  *KeyboardHandler
	debouncer *Debouncer
	config    *Config
}

// NewCommandHandlers creates a new CommandHandlers instance
func NewCommandHandlers(api *tgbotapi.BotAPI, logger *zap.Logger, debouncer *Debouncer, config *Config) *CommandHandlers {
	keyboard := NewKeyboardHandler()
	return &CommandHandlers{
		api:       api,
		logger:    logger,
		keyboard:  keyboard,
		debouncer: debouncer,
		config:    config,
	}
}

// SetBotCommands sets the bot's command menu using setMyCommands
func (h *CommandHandlers) SetBotCommands() error {
	commands := []tgbotapi.BotCommand{
		{
			Command:     "/month",
			Description: "Получить релизы за текущий месяц или выбрать месяц",
		},
		{
			Command:     "/whitelists",
			Description: "Показать вайтлисты",
		},
		{
			Command:     "/help",
			Description: "Показать это сообщение",
		},
	}

	config := tgbotapi.NewSetMyCommands(commands...)
	_, err := h.api.Request(config)
	if err != nil {
		h.logger.Error("Failed to set bot commands", zap.Error(err))
		return err
	}
	return nil
}

// HandleCommand processes incoming Telegram commands
func (h *CommandHandlers) HandleCommand(update tgbotapi.Update) {
	command := update.Message.Command()
	args := update.Message.CommandArguments()
	chatID := update.Message.Chat.ID

	switch command {
	case "start":
		h.handleStart(chatID)

	case "help":
		h.handleHelp(chatID)

	case "whitelists":
		h.handleWhitelists(chatID)

	case "month":
		h.handleMonth(chatID, args)

	case "clearcache":
		h.handleClearCache(update)

	default:
		h.handleDefault(chatID)
	}
}

// handleStart processes the /start command
func (h *CommandHandlers) handleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Добро пожаловать! Выберите месяц:")
	msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
	h.api.Send(msg)
}

// handleHelp processes the /help command
func (h *CommandHandlers) handleHelp(chatID int64) {
	helpText := "Парсю сайтик и шлю релизики\n\n" +
		"Доступные команды:\n" +
		"/help - показать это сообщение\n" +
		"/month - показать клавиатуру для выбора месяца\n" +
		"/month <month> - получить релизы за указанный месяц (например, /month march)\n" +
		"Параметр -gg: использовать только female_whitelist (например, /month march -gg)\n" +
		"Параметр -mg: использовать только male_whitelist (например, /month march -mg)\n" +
		"/whitelists - показать вайтлисты\n\n" +
		"По вопросам вайтлистов, обращаться к @" + h.config.AdminUsername

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
	h.api.Send(msg)
}

// handleWhitelists processes the /whitelists command
func (h *CommandHandlers) handleWhitelists(chatID int64) {
	femaleList := strings.Trim(os.Getenv("female_whitelist"), `"`)
	maleList := strings.Trim(os.Getenv("male_whitelist"), `"`)

	femaleFormatted := strings.ReplaceAll(femaleList, ",", ", ")
	maleFormatted := strings.ReplaceAll(maleList, ",", ", ")

	helpText := "Женский вайтлист: " + femaleFormatted + "\n\n" +
		"Мужской вайтлист: "
	if maleList != "" {
		helpText += maleFormatted
	} else {
		helpText += "не задан"
	}

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
	h.api.Send(msg)
}

// fetchReleases fetches releases for the given months with the specified whitelist filters
func (h *CommandHandlers) fetchReleases(months []string, femaleOnly, maleOnly bool, chatID int64) ([]models.Release, error) {
	// Загружаем оба whitelist'а для возможной фильтрации
	fullWhitelist := utils.LoadWhitelist(false)

	// Проверка на пустой whitelist
	if len(fullWhitelist) == 0 {
		h.logger.Error("Whitelist is empty")
		return nil, fmt.Errorf("whitelist is empty")
	}

	// Проверка на некорректные элементы в whitelist
	for key := range fullWhitelist {
		if strings.TrimSpace(key) == "" {
			h.logger.Error("Whitelist contains invalid entries")
			return nil, fmt.Errorf("whitelist contains invalid entries")
		}
	}

	var whitelist map[string]struct{}
	if femaleOnly {
		whitelist = utils.LoadFemaleWhitelist()
		if len(whitelist) == 0 {
			h.logger.Error("Female whitelist is empty")
			return nil, fmt.Errorf("female whitelist is empty")
		}
	} else if maleOnly {
		whitelist = utils.LoadMaleWhitelist()
		if len(whitelist) == 0 {
			h.logger.Error("Male whitelist is empty")
			return nil, fmt.Errorf("male whitelist is empty")
		}
	} else {
		whitelist = fullWhitelist
	}

	// Проверка на некорректные элементы в выбранном whitelist
	for key := range whitelist {
		if strings.TrimSpace(key) == "" {
			h.logger.Error("Selected whitelist contains invalid entries")
			return nil, fmt.Errorf("selected whitelist contains invalid entries")
		}
	}

	releases, err := parser.GetReleasesForMonths(months, whitelist, femaleOnly, maleOnly, fullWhitelist, h.logger)
	if err != nil {
		return nil, err
	}

	return releases, nil
}

// handleMonth processes the /month command
func (h *CommandHandlers) handleMonth(chatID int64, args string) {
	argsParts := strings.Fields(args)
	femaleOnly := false
	maleOnly := false
	month := ""

	for _, part := range argsParts {
		if part == "-gg" {
			femaleOnly = true
		} else if part == "-mg" {
			maleOnly = true
		} else {
			month = part
		}
	}

	// Проверяем, что не указаны оба флага одновременно
	if femaleOnly && maleOnly {
		msg := tgbotapi.NewMessage(chatID, "Ошибка: нельзя использовать -gg и -mg одновременно.")
		msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
		h.api.Send(msg)
		return
	}

	// Если месяц указан, выполняем запрос
	if month != "" {
		months := []string{month}
		releases, err := h.fetchReleases(months, femaleOnly, maleOnly, chatID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка получения релизов: %v", err))
			msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
			h.api.Send(msg)
			return
		}

		if len(releases) == 0 {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Релизы для %s не найдены. Проверьте whitelist или данные на сайте.", month))
			msg.ParseMode = "HTML"
			msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
			h.api.Send(msg)
			return
		}

		// Форматируем ответ
		var response strings.Builder
		for _, release := range releases {
			response.WriteString(formatReleaseForTelegram(release))
			response.WriteString("\n")
		}

		msg := tgbotapi.NewMessage(chatID, response.String())
		msg.ParseMode = "HTML"
		msg.DisableWebPagePreview = true
		msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
		h.api.Send(msg)
		return
	}

	// Если месяц не указан, показываем Inline Keyboard с месяцами
	msg := tgbotapi.NewMessage(chatID, "Выберите месяц:")
	msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
	h.api.Send(msg)
}

// handleClearCache processes the /clearcache command
func (h *CommandHandlers) handleClearCache(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	username := update.Message.From.UserName

	if username != h.config.AdminUsername {
		msg := tgbotapi.NewMessage(chatID, "У вас нет доступа к этой команде.")
		msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
		h.api.Send(msg)
		return
	}

	// Проверяем, пуст ли кэш
	cacheSize := len(parser.GetCacheKeys())
	if cacheSize == 0 {
		msg := tgbotapi.NewMessage(chatID, "Кэш уже пуст.")
		msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
		h.api.Send(msg)
		return
	}

	// Очищаем кэш
	parser.ClearCache()

	// Запускаем асинхронное обновление кэша
	go parser.InitializeCache(h.logger)

	// Сообщаем пользователю
	msg := tgbotapi.NewMessage(chatID, "Кэш очищен. Обновление кэша запущено.")
	msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
	h.api.Send(msg)
}

// handleDefault processes unknown commands
func (h *CommandHandlers) handleDefault(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте /help для списка доступных команд.")
	msg.ReplyMarkup = h.keyboard.GetMainMonthKeyboard()
	h.api.Send(msg)
}

// formatReleaseForTelegram formats a single release for Telegram message
func formatReleaseForTelegram(release models.Release) string {
	artist := html.EscapeString(release.Artist)
	albumName := html.EscapeString(release.AlbumName)
	albumName = strings.TrimPrefix(albumName, "Album: ")
	albumName = strings.TrimPrefix(albumName, "OST: ")
	cleanedTitleTrack := strings.ReplaceAll(release.TitleTrack, "Title Track:", "")
	cleanedTitleTrack = strings.TrimSpace(cleanedTitleTrack)
	trackName := html.EscapeString(cleanedTitleTrack)

	result := strings.Builder{}
	result.WriteString(release.Date + " | <b>" + artist + "</b>")

	if release.AlbumName != "N/A" {
		result.WriteString(" | " + albumName)
	}

	if release.MV != "" && release.MV != "N/A" {
		if trackName != "N/A" {
			result.WriteString(" | <a href=\"" + release.MV + "\">" + trackName + "</a>")
		} else {
			result.WriteString(" | <a href=\"" + release.MV + "\">" + "Link" + "</a>")
		}
	} else if trackName != "N/A" {
		result.WriteString(" | " + trackName)
	}

	return result.String()
}
