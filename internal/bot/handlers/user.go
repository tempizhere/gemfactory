package commands

import (
	"fmt"
	"gemfactory/internal/bot/middleware"
	"gemfactory/internal/bot/router"
	"gemfactory/internal/domain/types"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"
)

// RegisterUserRoutes registers user command handlers
func RegisterUserRoutes(r *router.Router, _ *types.Dependencies) {
	r.Handle("start", handleStart)
	r.Handle("help", handleHelp)
	r.Handle("month", middleware.Wrap(middleware.Debounce, handleMonth))
	r.Handle("whitelists", handleWhitelists)
	r.Handle("metrics", handleMetricsCommand)
	r.Handle("homework", handleHomework)
	r.Handle("playlist", handlePlaylist)
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
		"/playlist - Информация о плейлисте\n" +
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
	response.WriteString("📊 Метрики системы\n\n")

	// Пользовательская активность
	userActivity := stats["user_activity"].(map[string]interface{})
	response.WriteString("👥 Пользовательская активность:\n")
	response.WriteString(fmt.Sprintf("  • Всего команд: %v\n", userActivity["total_commands"]))
	response.WriteString(fmt.Sprintf("  • Уникальных пользователей: %v\n\n", userActivity["unique_users"]))

	// Артисты
	artists := stats["artists"].(map[string]interface{})
	response.WriteString("🎤 Артисты в фильтрах:\n")
	response.WriteString(fmt.Sprintf("  • Женские группы: %v\n", artists["female_artists"]))
	response.WriteString(fmt.Sprintf("  • Мужские группы: %v\n", artists["male_artists"]))
	response.WriteString(fmt.Sprintf("  • Всего артистов: %v\n\n", artists["total_artists"]))

	// Релизы
	releases := stats["releases"].(map[string]interface{})
	response.WriteString("💿 Релизы в кэше:\n")
	response.WriteString(fmt.Sprintf("  • Количество релизов: %v\n", releases["cached_releases"]))
	response.WriteString(fmt.Sprintf("  • Hit rate кэша: %.1f%%\n", releases["cache_hit_rate"]))
	response.WriteString(fmt.Sprintf("  • Попадания/промахи: %v/%v\n\n", releases["cache_hits"], releases["cache_misses"]))

	// Производительность
	performance := stats["performance"].(map[string]interface{})
	response.WriteString("⚡ Производительность:\n")
	response.WriteString(fmt.Sprintf("  • Среднее время ответа: %v\n", performance["avg_response_time"]))
	response.WriteString(fmt.Sprintf("  • Всего запросов: %v\n", performance["total_requests"]))
	response.WriteString(fmt.Sprintf("  • Ошибок: %v (%.1f%%)\n\n", performance["error_count"], performance["error_rate"]))

	// Система
	system := stats["system"].(map[string]interface{})
	response.WriteString("🔄 Статус системы:\n")
	response.WriteString(fmt.Sprintf("  • Время работы: %v\n", system["uptime"]))
	response.WriteString(fmt.Sprintf("  • Последнее обновление релизов: %v\n", system["last_cache_update"]))
	response.WriteString(fmt.Sprintf("  • Следующее обновление релизов: %v\n", system["next_cache_update"]))

	// Плейлист
	if ctx.Deps.PlaylistScheduler != nil {
		lastUpdate := ctx.Deps.PlaylistScheduler.GetLastUpdateTime()
		nextUpdate := ctx.Deps.PlaylistScheduler.GetNextUpdateTime()

		response.WriteString("\n🎵 Плейлист:\n")
		if ctx.Deps.PlaylistManager.IsLoaded() {
			response.WriteString(fmt.Sprintf("  • Статус: Загружен (%d треков)\n", ctx.Deps.PlaylistManager.GetTotalTracks()))
		} else {
			response.WriteString("  • Статус: Не загружен\n")
		}
		if !lastUpdate.IsZero() {
			response.WriteString(fmt.Sprintf("  • Последнее обновление: %s\n", lastUpdate.Format("02.01.06 15:04")))
			response.WriteString(fmt.Sprintf("  • Следующее обновление: %s\n", nextUpdate.Format("02.01.06 15:04")))
		} else {
			response.WriteString("  • Планировщик не запущен\n")
		}
	} else {
		response.WriteString("\n🎵 Плейлист:\n")
		response.WriteString("  • Планировщик не настроен\n")
	}

	// Домашние задания
	response.WriteString("\n📚 Домашние задания:\n")
	response.WriteString(fmt.Sprintf("  • Всего выдано заданий: %d\n", ctx.Deps.HomeworkCache.GetTotalRequests()))
	response.WriteString(fmt.Sprintf("  • Уникальных пользователей: %d\n", ctx.Deps.HomeworkCache.GetUniqueUsers()))

	return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, response.String())
}

// handleHomework обрабатывает команду /homework
func handleHomework(ctx types.Context) error {
	startTime := time.Now()
	userID := ctx.Message.From.ID

	ctx.Deps.Logger.Info("Homework command received",
		zap.String("command", "/homework"),
		zap.Int64("user_id", userID),
		zap.String("username", ctx.Message.From.UserName),
		zap.String("service", "telegram_bot"),
		zap.String("component", "homework_handler"))

	canRequest := ctx.Deps.HomeworkCache.CanRequest(userID)
	ctx.Deps.Logger.Debug("Homework request check result",
		zap.Int64("user_id", userID),
		zap.Bool("can_request", canRequest))

	if !canRequest {
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

			// Создаем ссылку на Spotify для текущего задания
			spotifyLink := fmt.Sprintf("https://open.spotify.com/track/%s", homeworkInfo.Track.ID)

			currentHomework = fmt.Sprintf("\n\n📚 Ваше текущее задание:\n🎵 \"%s - %s\" (<a href=\"%s\">Spotify</a>) %d %s",
				homeworkInfo.Track.Artist, homeworkInfo.Track.Title, spotifyLink, homeworkInfo.PlayCount, timesWord)
		}

		// Логируем отказ в выдаче домашнего задания
		duration := time.Since(startTime)
		ctx.Deps.Logger.Info("Homework request denied - already received today",
			zap.String("command", "/homework"),
			zap.Int64("user_id", userID),
			zap.String("username", ctx.Message.From.UserName),
			zap.Duration("duration", duration),
			zap.String("result", "denied"),
			zap.Duration("time_until_next", timeUntilNext),
			zap.String("service", "telegram_bot"),
			zap.String("component", "homework_handler"))

		return ctx.Deps.BotAPI.SendMessageWithReplyAndMarkup(ctx.Message.Chat.ID,
			fmt.Sprintf("⏰ Вы уже получили домашнее задание сегодня! Следующее задание будет доступно через %s.%s", timeMessage, currentHomework), ctx.Message.MessageID, nil)
	}

	if !ctx.Deps.PlaylistManager.IsLoaded() {
		duration := time.Since(startTime)
		ctx.Deps.Logger.Error("Homework request failed - playlist not loaded",
			zap.String("command", "/homework"),
			zap.Int64("user_id", userID),
			zap.String("username", ctx.Message.From.UserName),
			zap.Duration("duration", duration),
			zap.String("result", "error"),
			zap.String("error", "playlist_not_loaded"),
			zap.String("service", "telegram_bot"),
			zap.String("component", "homework_handler"))

		return ctx.Deps.BotAPI.SendMessageWithReply(ctx.Message.Chat.ID,
			"❌ Плейлист не загружен. Обратитесь к администратору для загрузки плейлиста.", ctx.Message.MessageID)
	}

	track, err := ctx.Deps.PlaylistManager.GetRandomTrack()
	if err != nil {
		duration := time.Since(startTime)
		ctx.Deps.Logger.Error("Homework request failed - failed to get random track",
			zap.String("command", "/homework"),
			zap.Int64("user_id", userID),
			zap.String("username", ctx.Message.From.UserName),
			zap.Duration("duration", duration),
			zap.String("result", "error"),
			zap.String("error", "failed_to_get_track"),
			zap.Error(err),
			zap.String("service", "telegram_bot"),
			zap.String("component", "homework_handler"))

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

	// Логируем успешное выполнение
	duration := time.Since(startTime)
	ctx.Deps.Logger.Info("Homework request processed",
		zap.String("command", "/homework"),
		zap.Int64("user_id", userID),
		zap.String("username", ctx.Message.From.UserName),
		zap.Duration("duration", duration),
		zap.String("result", "success"),
		zap.String("track_id", track.ID),
		zap.String("track_title", track.Title),
		zap.String("track_artist", track.Artist),
		zap.Int("play_count", playCount),
		zap.String("service", "telegram_bot"),
		zap.String("component", "homework_handler"))

	// Отправляем сообщение с reply к исходному сообщению
	return ctx.Deps.BotAPI.SendMessageWithReplyAndMarkup(ctx.Message.Chat.ID, message, ctx.Message.MessageID, nil)
}

// handlePlaylist обрабатывает команду /playlist
func handlePlaylist(ctx types.Context) error {
	if !ctx.Deps.PlaylistManager.IsLoaded() {
		return ctx.Deps.BotAPI.SendMessageWithReply(ctx.Message.Chat.ID,
			"❌ Плейлист не загружен. Обратитесь к администратору для загрузки плейлиста.", ctx.Message.MessageID)
	}

	// Получаем информацию о плейлисте
	playlistInfo, err := ctx.Deps.PlaylistManager.GetPlaylistInfo()
	if err != nil {
		return ctx.Deps.BotAPI.SendMessageWithReply(ctx.Message.Chat.ID,
			"❌ Ошибка при получении информации о плейлисте. Попробуйте позже.", ctx.Message.MessageID)
	}

	// Создаем ссылку на Spotify плейлист
	spotifyPlaylistLink := fmt.Sprintf("https://open.spotify.com/playlist/%s", playlistInfo.ID)

	// Формируем сообщение с информацией о плейлисте
	message := fmt.Sprintf("📚 Информация о плейлисте:\n\n"+
		"🎵 Название: %s\n"+
		"📊 Количество треков: %d\n"+
		"👤 Владелец: %s\n"+
		"📝 Описание: %s\n\n"+
		"🔗 Ссылка: (<a href=\"%s\">Открыть в Spotify</a>)",
		playlistInfo.Name, playlistInfo.TotalTracks, playlistInfo.Owner, playlistInfo.Description, spotifyPlaylistLink)

	return ctx.Deps.BotAPI.SendMessageWithReplyAndMarkup(ctx.Message.Chat.ID, message, ctx.Message.MessageID, nil)
}
