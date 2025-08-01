package commands

import (
	"fmt"
	"gemfactory/internal/bot/middleware"
	"gemfactory/internal/bot/router"
	"gemfactory/internal/domain/types"
	"math/rand"
	"strings"
)

// RegisterUserRoutes registers user command handlers
func RegisterUserRoutes(r *router.Router, _ *types.Dependencies) {
	r.Handle("start", handleStart)
	r.Handle("help", handleHelp)
	r.Handle("month", middleware.Wrap(middleware.Debounce, handleMonth))
	r.Handle("whitelists", handleWhitelists)
	r.Handle("metrics", handleMetricsCommand)
	r.Handle("homework", handleHomework)
}

func handleStart(ctx types.Context) error {
	return ctx.Deps.BotAPI.SendMessageWithMarkup(ctx.Message.Chat.ID, "Добро пожаловать! Выберите месяц:", ctx.Deps.Keyboard.GetMainKeyboard())
}

func handleHelp(ctx types.Context) error {
	text := "Доступные команды:\n" +
		"\n/start - Начать работу с ботом\n" +
		"/help - Показать это сообщение\n" +
		"/month [месяц] - Получить релизы за указанный месяц\n" +
		"/month [месяц] -gg - Релизы только женских групп\n" +
		"/month [месяц] -mg - Релизы только мужских групп\n" +
		"/whitelists - Показать списки артистов\n" +
		"/metrics - Показать метрики системы\n" +
		"/homework - Получить случайное домашнее задание\n" +
		"\n" +
		fmt.Sprintf("По вопросам вайтлистов: @%s", ctx.Deps.Config.GetAdminUsername())
	return ctx.Deps.BotAPI.SendMessageWithMarkup(ctx.Message.Chat.ID, text, ctx.Deps.Keyboard.GetMainKeyboard())
}

func handleMonth(ctx types.Context) error {
	args := strings.Fields(ctx.Message.Text)[1:]
	if len(args) == 0 {
		return ctx.Deps.BotAPI.SendMessageWithMarkup(ctx.Message.Chat.ID, "Пожалуйста, выберите месяц:", ctx.Deps.Keyboard.GetMainKeyboard())
	}

	month := strings.ToLower(args[0])
	femaleOnly := false
	maleOnly := false

	for _, arg := range args[1:] {
		switch arg {
		case "-gg":
			femaleOnly = true
		case "-mg":
			maleOnly = true
		}
	}

	response, err := ctx.Deps.ReleaseService.GetReleasesForMonth(month, femaleOnly, maleOnly)
	if err != nil {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, fmt.Sprintf("Ошибка: %v", err))
	}

	return ctx.Deps.BotAPI.SendMessageWithMarkup(ctx.Message.Chat.ID, response, ctx.Deps.Keyboard.GetMainKeyboard())
}

func handleWhitelists(ctx types.Context) error {
	response := ctx.Deps.ArtistService.FormatWhitelists()
	return ctx.Deps.BotAPI.SendMessageWithMarkup(ctx.Message.Chat.ID, response, ctx.Deps.Keyboard.GetMainKeyboard())
}

// handleMetricsCommand handles the /metrics command
func handleMetricsCommand(ctx types.Context) error {
	// Обновляем метрики перед отображением
	ctx.Deps.Metrics.UpdateArtistMetrics(
		len(ctx.Deps.ArtistService.GetFemaleWhitelist()),
		len(ctx.Deps.ArtistService.GetMaleWhitelist()),
	)
	ctx.Deps.Metrics.UpdateReleaseMetrics(ctx.Deps.Cache.GetCachedReleasesCount())

	stats := ctx.Deps.Metrics.GetStats()

	var response strings.Builder
	response.WriteString("📊 **Метрики системы**\n\n")

	// Пользовательская активность
	userActivity := stats["user_activity"].(map[string]interface{})
	response.WriteString("👥 **Пользовательская активность:**\n")
	response.WriteString(fmt.Sprintf("  • Всего команд: %v\n", userActivity["total_commands"]))
	response.WriteString(fmt.Sprintf("  • Уникальных пользователей: %v\n\n", userActivity["unique_users"]))

	// Артисты
	artists := stats["artists"].(map[string]interface{})
	response.WriteString("🎤 **Артисты в фильтрах:**\n")
	response.WriteString(fmt.Sprintf("  • Женские группы: %v\n", artists["female_artists"]))
	response.WriteString(fmt.Sprintf("  • Мужские группы: %v\n", artists["male_artists"]))
	response.WriteString(fmt.Sprintf("  • Всего артистов: %v\n\n", artists["total_artists"]))

	// Релизы
	releases := stats["releases"].(map[string]interface{})
	response.WriteString("💿 **Релизы в кэше:**\n")
	response.WriteString(fmt.Sprintf("  • Количество релизов: %v\n", releases["cached_releases"]))
	response.WriteString(fmt.Sprintf("  • Hit rate кэша: %.1f%%\n", releases["cache_hit_rate"]))
	response.WriteString(fmt.Sprintf("  • Попадания/промахи: %v/%v\n\n", releases["cache_hits"], releases["cache_misses"]))

	// Производительность
	performance := stats["performance"].(map[string]interface{})
	response.WriteString("⚡ **Производительность:**\n")
	response.WriteString(fmt.Sprintf("  • Среднее время ответа: %v\n", performance["avg_response_time"]))
	response.WriteString(fmt.Sprintf("  • Всего запросов: %v\n", performance["total_requests"]))
	response.WriteString(fmt.Sprintf("  • Ошибок: %v (%.1f%%)\n\n", performance["error_count"], performance["error_rate"]))

	// Система
	system := stats["system"].(map[string]interface{})
	response.WriteString("🔄 **Статус системы:**\n")
	response.WriteString(fmt.Sprintf("  • Время работы: %v\n", system["uptime"]))
	response.WriteString(fmt.Sprintf("  • Последнее обновление: %v\n", system["last_cache_update"]))
	response.WriteString(fmt.Sprintf("  • Следующее обновление: %v\n", system["next_cache_update"]))

	// Домашние задания
	response.WriteString("\n📚 **Домашние задания:**\n")
	response.WriteString(fmt.Sprintf("  • Всего выдано заданий: %d\n", ctx.Deps.HomeworkCache.GetTotalRequests()))
	response.WriteString(fmt.Sprintf("  • Уникальных пользователей: %d\n", ctx.Deps.HomeworkCache.GetUniqueUsers()))

	return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, response.String())
}

// handleHomework обрабатывает команду /homework
func handleHomework(ctx types.Context) error {
	userID := ctx.Message.From.ID

	if !ctx.Deps.HomeworkCache.CanRequest(userID) {
		timeUntilNext := ctx.Deps.HomeworkCache.GetTimeUntilNextRequest(userID)
		hours := int(timeUntilNext.Hours())
		minutes := int(timeUntilNext.Minutes()) % 60

		var timeMessage string
		if hours > 0 {
			timeMessage = fmt.Sprintf("%d ч %d мин", hours, minutes)
		} else {
			timeMessage = fmt.Sprintf("%d мин", minutes)
		}

		// Получаем информацию о текущем домашнем задании
		homeworkInfo := ctx.Deps.HomeworkCache.GetHomeworkInfo(userID)
		var currentHomework string
		if homeworkInfo != nil && homeworkInfo.Track != nil {
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

			currentHomework = fmt.Sprintf("\n\n📚 Ваше текущее задание:\n🎵 \"%s - %s\" %d %s",
				homeworkInfo.Track.Artist, homeworkInfo.Track.Title, homeworkInfo.PlayCount, timesWord)
		}

		return ctx.Deps.BotAPI.SendMessageWithReply(ctx.Message.Chat.ID,
			fmt.Sprintf("⏰ Вы уже получили домашнее задание сегодня! Следующее задание будет доступно через %s.%s", timeMessage, currentHomework), ctx.Message.MessageID)
	}

	if !ctx.Deps.PlaylistManager.IsLoaded() {
		return ctx.Deps.BotAPI.SendMessageWithReply(ctx.Message.Chat.ID,
			"❌ Плейлист не загружен. Обратитесь к администратору для загрузки плейлиста.", ctx.Message.MessageID)
	}

	track, err := ctx.Deps.PlaylistManager.GetRandomTrack()
	if err != nil {
		return ctx.Deps.BotAPI.SendMessageWithReply(ctx.Message.Chat.ID,
			"❌ Ошибка при получении трека из плейлиста. Попробуйте позже.", ctx.Message.MessageID)
	}

	// Генерируем случайное число от 1 до 6
	playCount := rand.Intn(6) + 1

	// Пул эмодзи для случайного выбора
	musicEmojis := []string{"🎵", "🎶", "🎼", "🎤", "🎸", "🎹", "🎺", "🎻", "🥁", "🎷"}
	selectedMusicEmoji := musicEmojis[rand.Intn(len(musicEmojis))]

	// Создаем ссылку на Spotify для встраивания в текст
	spotifyLink := fmt.Sprintf("https://open.spotify.com/track/%s", track.ID)

	// Правильное склонение для "раз/раза/раз"
	var timesWord string
	switch {
	case playCount == 1:
		timesWord = "раз"
	case playCount >= 2 && playCount <= 4:
		timesWord = "раза"
	default:
		timesWord = "раз"
	}

	// Формируем сообщение с кликабельным Spotify в скобках
	message := fmt.Sprintf("🎲 Ваше домашнее задание: послушать \"%s - %s\" (<a href=\"%s\">Spotify</a>) %s %d %s",
		track.Artist, track.Title, spotifyLink, selectedMusicEmoji, playCount, timesWord)

	// Записываем запрос в кэш
	ctx.Deps.HomeworkCache.RecordRequest(userID, track, playCount)

	// Отправляем сообщение с reply к исходному сообщению
	return ctx.Deps.BotAPI.SendMessageWithReplyAndMarkup(ctx.Message.Chat.ID, message, ctx.Message.MessageID, nil)
}
