// Package handlers содержит обработчики административных команд.
package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// AddArtist добавляет артистов в whitelist
func (h *Handlers) AddArtist(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	args := strings.Fields(message.CommandArguments())
	if len(args) < 2 {
		h.sendMessage(message.Chat.ID, "Использование: /add_artist <artist_names> [-f|-m]\nПример: /add_artist ITZY -f\nПример множественных: /add_artist ablume, aespa, apink -f")
		return
	}

	// Извлекаем имена артистов и флаг
	artistNamesStr := strings.Join(args[:len(args)-1], " ")
	flag := strings.ToLower(args[len(args)-1])

	var isFemale bool
	var genderFlag string

	// Проверяем флаги
	switch flag {
	case "-f":
		isFemale = true
		genderFlag = "женский"
	case "-m":
		isFemale = false
		genderFlag = "мужской"
	default:
		h.sendMessage(message.Chat.ID, "Флаг должен быть -f (женский) или -m (мужской). Пример: /add_artist ITZY -f")
		return
	}

	// Парсим имена артистов (разделенные запятыми)
	artistNames := h.parseArtists(artistNamesStr)
	if len(artistNames) == 0 {
		h.sendMessage(message.Chat.ID, "Не найдено ни одного имени артиста")
		return
	}

	addedCount, err := h.services.Artist.AddArtists(artistNames, isFemale)
	if err != nil {
		h.logger.Error("Failed to add artists", zap.Error(err))
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Ошибка при добавлении артистов: %v", err))
		return
	}

	if addedCount == 0 {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Все артисты уже существуют в списке: %s", strings.Join(artistNames, ", ")))
		return
	}

	// Формируем сообщение о результате
	if len(artistNames) == 1 {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Добавлен %s артист: %s", genderFlag, artistNames[0]))
	} else {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Добавлено %d %s артистов из %d: %s",
			addedCount, genderFlag, len(artistNames), strings.Join(artistNames, ", ")))
	}
}

// RemoveArtist деактивирует артистов (снимает флаг is_active)
func (h *Handlers) RemoveArtist(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	args := strings.Fields(message.CommandArguments())
	if len(args) < 1 {
		h.sendMessage(message.Chat.ID, "Использование: /remove_artist <artist_names>\nПример: /remove_artist ITZY\nПример множественных: /remove_artist ablume, aespa, apink")
		return
	}

	// Объединяем все аргументы в одну строку для парсинга
	artistNamesStr := strings.Join(args, " ")

	// Парсим имена артистов (разделенные запятыми)
	artistNames := h.parseArtists(artistNamesStr)
	if len(artistNames) == 0 {
		h.sendMessage(message.Chat.ID, "Не найдено ни одного имени артиста")
		return
	}

	deactivatedCount, err := h.services.Artist.DeactivateArtists(artistNames)
	if err != nil {
		h.logger.Error("Failed to deactivate artists", zap.Error(err))
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Ошибка при деактивации артистов: %v", err))
		return
	}

	if deactivatedCount == 0 {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Артисты не найдены или уже деактивированы: %s", strings.Join(artistNames, ", ")))
		return
	}

	// Формируем сообщение о результате
	if len(artistNames) == 1 {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Артист %s деактивирован (исключен из парсинга и отображения)", artistNames[0]))
	} else {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Деактивировано %d артистов из %d: %s",
			deactivatedCount, len(artistNames), strings.Join(artistNames, ", ")))
	}
}

// ClearWhitelists очищает все whitelist
func (h *Handlers) ClearWhitelists(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	// Получаем всех артистов и удаляем их
	artists, err := h.services.Artist.GetAll()
	if err != nil {
		h.logger.Error("Failed to get artists", zap.Error(err))
		h.sendMessage(message.Chat.ID, "❌ Ошибка при получении списка артистов.")
		return
	}

	var artistNames []string
	for _, artist := range artists {
		artistNames = append(artistNames, artist.Name)
	}

	if len(artistNames) > 0 {
		_, err = h.services.Artist.RemoveArtists(artistNames)
		if err != nil {
			h.logger.Error("Failed to remove artists", zap.Error(err))
			h.sendMessage(message.Chat.ID, "❌ Ошибка при удалении артистов.")
			return
		}
	}

	h.sendMessage(message.Chat.ID, "✅ Все артисты удалены.")
}

// ClearCache очищает кэш релизов
func (h *Handlers) ClearCache(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	h.sendMessage(message.Chat.ID, "✅ Кэш очищен, обновление запущено")
}

// Export экспортирует данные
func (h *Handlers) Export(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	// Экспорт данных
	response, err := h.services.Artist.Export()
	if err != nil {
		h.logger.Error("Failed to export artists", zap.Error(err))
		h.sendMessage(message.Chat.ID, "❌ Ошибка при экспорте данных.")
		return
	}
	h.sendMessageWithMarkup(message.Chat.ID, response, h.getMainKeyboard())
}

// Config устанавливает конфигурацию
func (h *Handlers) Config(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	args := strings.Fields(message.CommandArguments())
	if len(args) != 2 {
		h.sendMessage(message.Chat.ID, "Использование: /config [key] [value]")
		return
	}

	key := args[0]
	value := args[1]

	err := h.services.Config.Set(key, value)
	if err != nil {
		h.logger.Error("Failed to set config", zap.Error(err))
		h.sendMessage(message.Chat.ID, "Ошибка при установке конфигурации")
		return
	}

	h.sendMessage(message.Chat.ID, fmt.Sprintf("Конфигурация %s установлена в %s", key, value))
}

// ConfigList показывает текущую конфигурацию
func (h *Handlers) ConfigList(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	config, err := h.services.Config.GetAll()
	if err != nil {
		h.logger.Error("Failed to get config", zap.Error(err))
		h.sendMessage(message.Chat.ID, "Ошибка при получении конфигурации")
		return
	}

	h.sendMessage(message.Chat.ID, config)
}

// ConfigReset сбрасывает конфигурацию
func (h *Handlers) ConfigReset(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	err := h.services.Config.Reset()
	if err != nil {
		h.logger.Error("Failed to reset config", zap.Error(err))
		h.sendMessage(message.Chat.ID, "Ошибка при сбросе конфигурации")
		return
	}

	h.sendMessage(message.Chat.ID, "Конфигурация сброшена к значениям по умолчанию")
}

// ParseReleases парсит релизы за указанный период
func (h *Handlers) ParseReleases(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	args := strings.Fields(message.CommandArguments())

	// Если аргументы не указаны, парсим текущий месяц
	if len(args) == 0 {
		currentMonth := strings.ToLower(time.Now().Format("January"))
		currentYear := time.Now().Year()
		h.logger.Info("No arguments provided, parsing current month",
			zap.String("month", currentMonth),
			zap.Int("year", currentYear))

		// Отправляем сообщение о начале парсинга
		h.sendMessage(message.Chat.ID, fmt.Sprintf("🔄 Начинаю парсинг релизов за %s %d...", currentMonth, currentYear))

		// Запускаем парсинг в горутине
		go func() {
			ctx := context.Background()
			totalCount, err := h.parseMonth(ctx, currentMonth, currentYear)

			if err != nil {
				h.logger.Error("Failed to parse releases", zap.Error(err))
				h.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка при парсинге релизов: %v", err))
				return
			}

			h.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Парсинг завершен! Сохранено %d релизов за %s %d", totalCount, currentMonth, currentYear))
		}()
		return
	}

	// Отправляем сообщение о начале парсинга
	h.sendMessage(message.Chat.ID, "🔄 Начинаю парсинг релизов...")

	// Запускаем парсинг в горутине
	go func() {
		ctx := context.Background()
		var totalCount int
		var err error

		if len(args) == 1 {
			// Проверяем, является ли аргумент годом (4 цифры)
			if year, parseErr := strconv.Atoi(args[0]); parseErr == nil && year >= 2000 && year <= 2100 {
				// Парсинг всего года
				totalCount, err = h.parseYear(ctx, year)
			} else {
				// Парсинг месяца текущего года
				month := strings.ToLower(args[0])
				currentYear := time.Now().Year()
				totalCount, err = h.parseMonth(ctx, month, currentYear)
			}
		} else if len(args) == 2 {
			// Парсинг конкретного месяца и года
			month := strings.ToLower(args[0])
			year, parseErr := strconv.Atoi(args[1])
			if parseErr != nil {
				h.sendMessage(message.Chat.ID, "❌ Неверный формат года. Используйте 4 цифры (например: 2025)")
				return
			}
			totalCount, err = h.parseMonth(ctx, month, year)
		} else {
			h.sendMessage(message.Chat.ID, "❌ Слишком много аргументов.\n\n"+
				"Использование:\n"+
				"• /parse_releases - парсинг текущего месяца\n"+
				"• /parse_releases <месяц> - парсинг месяца текущего года\n"+
				"• /parse_releases <месяц> <год> - парсинг конкретного месяца и года\n"+
				"• /parse_releases <год> - парсинг всего года\n\n"+
				"Примеры:\n"+
				"• /parse_releases\n"+
				"• /parse_releases september\n"+
				"• /parse_releases september 2025\n"+
				"• /parse_releases 2025")
			return
		}

		if err != nil {
			h.logger.Error("Failed to parse releases", zap.Error(err))
			h.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка при парсинге релизов: %v", err))
			return
		}

		h.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Парсинг завершен! Сохранено %d релизов", totalCount))
	}()
}

// parseMonth парсит релизы за конкретный месяц и год
func (h *Handlers) parseMonth(ctx context.Context, month string, year int) (int, error) {
	h.logger.Info("Parsing month", zap.String("month", month), zap.Int("year", year))

	// Формируем строку месяца с годом для скрейпера
	monthWithYear := fmt.Sprintf("%s-%d", month, year)

	count, err := h.services.Release.ParseReleasesForMonth(ctx, monthWithYear)
	if err != nil {
		return 0, fmt.Errorf("failed to parse month %s %d: %w", month, year, err)
	}

	return count, nil
}

// parseYear парсит релизы за весь год
func (h *Handlers) parseYear(ctx context.Context, year int) (int, error) {
	h.logger.Info("Parsing year", zap.Int("year", year))

	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	totalCount := 0
	for _, month := range months {
		monthWithYear := fmt.Sprintf("%s-%d", month, year)

		count, err := h.services.Release.ParseReleasesForMonth(ctx, monthWithYear)
		if err != nil {
			h.logger.Warn("Failed to parse month",
				zap.String("month", month),
				zap.Int("year", year),
				zap.Error(err))
			continue
		}

		totalCount += count
		h.logger.Info("Parsed month",
			zap.String("month", month),
			zap.Int("year", year),
			zap.Int("count", count))
	}

	return totalCount, nil
}

// parseArtists парсит список артистов из строки
func (h *Handlers) parseArtists(input string) []string {
	// Разделяем по запятым и очищаем от пробелов
	parts := strings.Split(input, ",")
	var artists []string
	for _, part := range parts {
		artist := strings.TrimSpace(part)
		if artist != "" {
			artists = append(artists, artist)
		}
	}
	return artists
}

// sendMessageWithMarkup отправляет сообщение с клавиатурой
func (h *Handlers) sendMessageWithMarkup(chatID int64, text string, markup interface{}) {
	if h.botAPI != nil {
		err := h.botAPI.SendMessageWithMarkup(chatID, text, markup)
		if err != nil {
			h.logger.Error("Failed to send message with markup", zap.Int64("chat_id", chatID), zap.Error(err))
		}
	} else {
		h.logger.Warn("BotAPI not available, cannot send message with markup", zap.Int64("chat_id", chatID))
	}
}

// getMainKeyboard возвращает основную клавиатуру
func (h *Handlers) getMainKeyboard() tgbotapi.InlineKeyboardMarkup {
	return h.keyboard.GetMainKeyboard()
}

// TasksList показывает список всех задач
func (h *Handlers) TasksList(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	tasks, err := h.services.Task.GetAllTasks()
	if err != nil {
		h.logger.Error("Failed to get tasks", zap.Error(err))
		h.sendMessage(message.Chat.ID, "Ошибка при получении списка задач")
		return
	}

	if len(tasks) == 0 {
		h.sendMessage(message.Chat.ID, "📋 Задачи не найдены")
		return
	}

	var result strings.Builder
	result.WriteString("📋 Список задач:\n\n")

	for _, task := range tasks {
		// Статус активности
		status := "🔴 Неактивна"
		if task.IsActive {
			status = "🟢 Активна"
		}

		result.WriteString(fmt.Sprintf("🔧 <b>%s</b> (%s)\n", task.Name, status))
		result.WriteString(fmt.Sprintf("   📝 %s\n", task.Description))
		result.WriteString(fmt.Sprintf("   ⏰ Cron: %s\n", task.CronExpression))
		result.WriteString(fmt.Sprintf("   📊 Запусков: %d (успешно: %d, ошибок: %d)\n",
			task.RunCount, task.SuccessCount, task.ErrorCount))

		if task.LastRun != nil {
			result.WriteString(fmt.Sprintf("   🕐 Последний запуск: %s\n",
				task.LastRun.Format("02.01.2006 15:04:05")))
		}

		if task.NextRun != nil {
			result.WriteString(fmt.Sprintf("   ⏭️ Следующий запуск: %s\n",
				task.NextRun.Format("02.01.2006 15:04:05")))
		}

		if task.LastError != "" {
			result.WriteString(fmt.Sprintf("   ❌ Последняя ошибка: %s\n", task.LastError))
		}

		result.WriteString("\n")
	}

	h.sendMessage(message.Chat.ID, result.String())
}

// ReloadPlaylist перезагружает плейлист из Spotify
func (h *Handlers) ReloadPlaylist(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	h.sendMessage(message.Chat.ID, "🔄 Начинаю перезагрузку плейлиста...")

	err := h.services.Playlist.ReloadPlaylist()
	if err != nil {
		h.logger.Error("Failed to reload playlist", zap.Error(err))
		h.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка при перезагрузке плейлиста: %v", err))
		return
	}

	h.sendMessage(message.Chat.ID, "✅ Плейлист успешно перезагружен!")
}

// Admin обрабатывает команду /admin
func (h *Handlers) Admin(message *tgbotapi.Message) {
	// Проверка прав администратора
	if !h.isAdmin(message.From) {
		h.sendMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды")
		return
	}

	text := "🔧 <b>Команды администратора:</b>\n\n" +
		"/add_artist [имена] [-f|-m] - Добавить артиста(ов)\n" +
		"/remove_artist [имена] - Деактивировать артиста(ов)\n" +
		"/export - Экспорт всех артистов\n" +
		"/config [ключ] [значение] - Установить конфигурацию\n" +
		"/config_list - Показать конфигурацию\n" +
		"/config_reset - Сбросить конфигурацию\n" +
		"/tasks_list - Показать список задач\n" +
		"/reload_playlist - Перезагрузить плейлист\n" +
		"/parse_releases [год] - Парсинг релизов\n" +
		"/parse_releases [месяц] [год] - Парсинг конкретного месяца\n" +
		"/parse_releases [месяц] - Парсинг месяца текущего года\n" +
		"/parse_releases - Парсинг текущего месяца\n\n" +
		"<b>Примеры множественных артистов:</b>\n" +
		"/add_artist ablume, aespa, apink -f\n" +
		"/remove_artist ablume, aespa, apink"

	h.sendMessage(message.Chat.ID, text)
}

// sendMessage отправляет сообщение
func (h *Handlers) sendMessage(chatID int64, text string) {
	if h.botAPI != nil {
		err := h.botAPI.SendMessage(chatID, text)
		if err != nil {
			h.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.Error(err))
		}
	} else {
		h.logger.Warn("BotAPI not available, cannot send message", zap.Int64("chat_id", chatID))
	}
}
