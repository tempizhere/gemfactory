// Package handlers содержит обработчики пользовательских команд.
package handlers

import (
	"fmt"
	"math/rand"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Start обрабатывает команду /start
func (h *Handlers) Start(message *tgbotapi.Message) {
	text := "Добро пожаловать! Выберите месяц:"
	h.sendMessageWithMarkup(message.Chat.ID, text, h.getMainKeyboard())
}

// Help обрабатывает команду /help
func (h *Handlers) Help(message *tgbotapi.Message) {
	text := "Доступные команды:\n" +
		"\n/start - Начать работу с ботом\n" +
		"/help - Показать это сообщение\n" +
		"/month [месяц] - Получить релизы за указанный месяц текущего года\n" +
		"/month [месяц] [год] - Получить релизы за указанный месяц и год\n" +
		"/month [месяц] -f - Релизы только женских групп\n" +
		"/month [месяц] -m - Релизы только мужских групп\n" +
		"/search [артист] - Поиск релизов по артисту\n" +
		"/artists - Показать списки артистов\n" +
		"/metrics - Показать метрики системы\n" +
		"/homework - Получить случайное домашнее задание\n" +
		"/playlist - Информация о плейлисте\n" +
		"\n" +
		fmt.Sprintf("По вопросам вайтлистов: @%s", h.getAdminUsername())
	h.sendMessageWithMarkup(message.Chat.ID, text, h.getMainKeyboard())
}

// Month обрабатывает команду /month
func (h *Handlers) Month(message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())
	if len(args) == 0 {
		h.sendMessageWithMarkup(message.Chat.ID, "Пожалуйста, выберите месяц:", h.getMainKeyboard())
		return
	}

	month := strings.ToLower(args[0])
	femaleOnly := false
	maleOnly := false
	year := ""

	// Обрабатываем аргументы
	for i, arg := range args[1:] {
		switch arg {
		case "-f":
			femaleOnly = true
		case "-m":
			maleOnly = true
		default:
			// Если это не флаг, то это может быть год
			if year == "" && i == 0 {
				year = arg
			}
		}
	}

	// Формируем строку месяца для поиска
	monthQuery := month
	if year != "" {
		monthQuery = fmt.Sprintf("%s-%s", month, year)
	}

	response, err := h.services.Release.GetReleasesForMonth(monthQuery, femaleOnly, maleOnly)
	if err != nil {
		h.logger.Error("Failed to get releases", zap.Error(err))
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Ошибка: %v", err))
		return
	}

	h.sendMessageWithMarkup(message.Chat.ID, response, h.getMainKeyboard())
}

// Artists показывает списки артистов
func (h *Handlers) Artists(message *tgbotapi.Message) {
	response := h.services.Artist.FormatArtists()
	h.sendMessageWithMarkup(message.Chat.ID, response, h.getMainKeyboard())
}

// Metrics показывает метрики системы
func (h *Handlers) Metrics(message *tgbotapi.Message) {
	// TODO: Реализовать получение метрик
	text := "📊 Метрики системы\n\n" +
		"👥 Пользовательская активность:\n" +
		"  • Всего команд: 0\n" +
		"  • Уникальных пользователей: 0\n\n" +
		"🎤 Артисты в фильтрах:\n" +
		"  • Женские группы: 0\n" +
		"  • Мужские группы: 0\n" +
		"  • Всего артистов: 0\n\n" +
		"💿 Релизы в кэше:\n" +
		"  • Количество релизов: 0\n" +
		"  • Hit rate кэша: 0.0%\n" +
		"  • Попадания/промахи: 0/0\n\n" +
		"⚡ Производительность:\n" +
		"  • Среднее время ответа: 0ms\n" +
		"  • Всего запросов: 0\n" +
		"  • Ошибок: 0 (0.0%)\n\n" +
		"🔄 Статус системы:\n" +
		"  • Время работы: 0\n" +
		"  • Последнее обновление релизов: Не обновлялось\n" +
		"  • Следующее обновление релизов: Не запланировано\n\n" +
		"🎵 Плейлист:\n" +
		"  • Статус: Не загружен\n" +
		"  • Планировщик не настроен\n\n" +
		"📚 Домашние задания:\n" +
		"  • Всего выдано заданий: 0\n" +
		"  • Уникальных пользователей: 0"

	h.sendMessage(message.Chat.ID, text)
}

// Homework выдает случайное домашнее задание
func (h *Handlers) Homework(message *tgbotapi.Message) {
	userID := message.From.ID

	canRequest, err := h.services.Homework.CanRequestHomework(userID)
	if err != nil {
		h.logger.Error("Failed to check homework request", zap.Error(err))
		h.sendMessage(message.Chat.ID, "❌ Ошибка при проверке возможности запроса домашнего задания.")
		return
	}
	if !canRequest {
		timeUntilNext := h.services.Homework.GetTimeUntilNextRequest(userID)
		hours := int(timeUntilNext.Hours())
		minutes := int(timeUntilNext.Minutes()) % 60

		var timeMessage string
		if hours > 0 {
			timeMessage = fmt.Sprintf("%d ч %d мин", hours, minutes)
		} else {
			timeMessage = fmt.Sprintf("%d мин", minutes)
		}

		// Получаем информацию о текущем домашнем задании
		homeworkInfo, err := h.services.Homework.GetActiveHomework(userID)
		if err != nil {
			h.logger.Error("Failed to get active homework", zap.Error(err))
		}
		var currentHomework string
		if homeworkInfo != nil {
			// Формируем правильное склонение для "раз/раза"
			var timesWord string
			switch {
			case homeworkInfo.PlayCount == 1:
				timesWord = "раз"
			case homeworkInfo.PlayCount >= 2 && homeworkInfo.PlayCount <= 4:
				timesWord = "раза"
			default:
				timesWord = "раз"
			}

			// Создаем ссылку на Spotify для текущего задания
			spotifyLink := fmt.Sprintf("https://open.spotify.com/track/%s", homeworkInfo.TrackID)

			currentHomework = fmt.Sprintf("\n\n📚 Ваше текущее задание:\n🎵 \"%s - %s\" (<a href=\"%s\">Spotify</a>) %d %s",
				homeworkInfo.Artist, homeworkInfo.Title, spotifyLink, homeworkInfo.PlayCount, timesWord)
		}

		h.sendMessageWithReply(message.Chat.ID,
			fmt.Sprintf("⏰ Вы уже получили домашнее задание сегодня! Следующее задание будет доступно через %s.%s", timeMessage, currentHomework), message.MessageID)
		return
	}

	homework, err := h.services.Homework.GetRandomHomework(userID)
	if err != nil {
		h.logger.Error("Failed to get homework", zap.Error(err))
		h.sendMessageWithReply(message.Chat.ID, "❌ Ошибка при получении домашнего задания. Попробуйте позже.", message.MessageID)
		return
	}

	// Пул эмодзи для случайного выбора
	musicEmojis := []string{"🎵", "🎶", "🎼", "🎤", "🎸", "🎹", "🎺", "🎻", "🥁", "🎷"}
	selectedMusicEmoji := musicEmojis[rand.Intn(len(musicEmojis))]

	// Создаем ссылку на Spotify для встраивания в текст
	spotifyLink := fmt.Sprintf("https://open.spotify.com/track/%s", homework.TrackID)

	// Правильное склонение для "раз/раза/раз"
	var timesWord string
	switch {
	case homework.PlayCount == 1:
		timesWord = "раз"
	case homework.PlayCount >= 2 && homework.PlayCount <= 4:
		timesWord = "раза"
	default:
		timesWord = "раз"
	}

	// Формируем сообщение с кликабельным Spotify в скобках
	messageText := fmt.Sprintf("🎲 Ваше домашнее задание: послушать \"%s - %s\" (<a href=\"%s\">Spotify</a>) %s %d %s",
		homework.Artist, homework.Title, spotifyLink, selectedMusicEmoji, homework.PlayCount, timesWord)

	// Отправляем сообщение с reply к исходному сообщению
	h.sendMessageWithReplyAndMarkup(message.Chat.ID, messageText, message.MessageID, h.getMainKeyboard())
}

// Playlist показывает информацию о плейлисте
func (h *Handlers) Playlist(message *tgbotapi.Message) {
	// Получаем URL плейлиста из конфигурации
	playlistURL, err := h.services.Config.GetConfigValue("PLAYLIST_URL")
	if err != nil {
		h.logger.Error("Failed to get playlist URL from config", zap.Error(err))
		h.sendMessageWithReply(message.Chat.ID, "❌ Ошибка при получении URL плейлиста из конфигурации.", message.MessageID)
		return
	}

	if playlistURL == "" {
		h.sendMessageWithReply(message.Chat.ID, "❌ URL плейлиста не настроен в конфигурации.", message.MessageID)
		return
	}

	info, err := h.services.Playlist.GetPlaylistInfo()
	if err != nil {
		h.logger.Error("Failed to get playlist info", zap.Error(err))
		h.sendMessageWithReply(message.Chat.ID, "❌ Ошибка при получении информации о плейлисте. Попробуйте позже.", message.MessageID)
		return
	}

	// Создаем ссылку на Spotify плейлист
	spotifyPlaylistLink := fmt.Sprintf("https://open.spotify.com/playlist/%s", info.SpotifyID)

	// Формируем сообщение с информацией о плейлисте
	messageText := fmt.Sprintf("📚 Информация о плейлисте:\n\n"+
		"🎵 Название: %s\n"+
		"📊 Количество треков: %d\n"+
		"👤 Владелец: %s\n"+
		"📝 Описание: %s\n\n"+
		"🔗 Ссылка: (<a href=\"%s\">Открыть в Spotify</a>)",
		info.Name, info.TrackCount, info.Owner, info.Description, spotifyPlaylistLink)

	h.sendMessageWithReplyAndMarkup(message.Chat.ID, messageText, message.MessageID, h.getMainKeyboard())
}

// CallbackQuery обрабатывает callback query
func (h *Handlers) CallbackQuery(query *tgbotapi.CallbackQuery) {
	err := h.keyboard.HandleCallbackQuery(query)
	if err != nil {
		h.logger.Error("Failed to handle callback query", zap.Error(err), zap.String("data", query.Data))
	}
}

// Unknown обрабатывает неизвестные команды
func (h *Handlers) Unknown(message *tgbotapi.Message) {
	h.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /help для получения справки.")
}

// sendMessageWithReply отправляет сообщение с reply
func (h *Handlers) sendMessageWithReply(chatID int64, text string, replyToMessageID int) {
	if h.botAPI != nil {
		err := h.botAPI.SendMessageWithReply(chatID, text, replyToMessageID)
		if err != nil {
			h.logger.Error("Failed to send message with reply", zap.Int64("chat_id", chatID), zap.Error(err))
		}
	} else {
		h.logger.Warn("BotAPI not available, cannot send message with reply", zap.Int64("chat_id", chatID))
	}
}

// sendMessageWithReplyAndMarkup отправляет сообщение с reply и клавиатурой
func (h *Handlers) sendMessageWithReplyAndMarkup(chatID int64, text string, replyToMessageID int, markup interface{}) {
	if h.botAPI != nil {
		err := h.botAPI.SendMessageWithReplyAndMarkup(chatID, text, replyToMessageID, markup)
		if err != nil {
			h.logger.Error("Failed to send message with reply and markup", zap.Int64("chat_id", chatID), zap.Error(err))
		}
	} else {
		h.logger.Warn("BotAPI not available, cannot send message with reply and markup", zap.Int64("chat_id", chatID))
	}
}

// getAdminUsername возвращает имя администратора
func (h *Handlers) getAdminUsername() string {
	// Получаем username из конфигурации
	username, err := h.services.Config.GetConfigValue("ADMIN_USERNAME")
	if err != nil {
		h.logger.Warn("Failed to get admin username from config", zap.Error(err))
		return "admin" // Fallback значение
	}

	if username == "" {
		return "admin" // Fallback значение
	}

	return username
}

// Search обрабатывает команду /search
func (h *Handlers) Search(message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())
	if len(args) == 0 {
		h.sendMessage(message.Chat.ID, "Использование: /search имя_артиста\nПример: /search ITZY")
		return
	}

	// Объединяем все аргументы в одно имя артиста
	artistName := strings.Join(args, " ")

	// Получаем релизы по артисту
	response, err := h.services.Release.GetReleasesByArtistName(artistName)
	if err != nil {
		h.logger.Error("Failed to get releases by artist", zap.Error(err))
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Ошибка при поиске релизов: %v", err))
		return
	}

	// Отправляем ответ
	h.sendMessage(message.Chat.ID, response)
}
