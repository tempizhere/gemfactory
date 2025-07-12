package commands

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/router"
	"gemfactory/internal/telegrambot/bot/types"
	"strings"
)

// RegisterUserRoutes registers user command handlers
func RegisterUserRoutes(r *router.Router, deps *types.Dependencies) {
	r.Handle("start", handleStart)
	r.Handle("help", handleHelp)
	r.Handle("month", handleMonth)
	r.Handle("whitelists", handleWhitelists)
	r.Handle("metrics", handleMetricsCommand)
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
	var response strings.Builder
	response.WriteString("📊 **Метрики системы**\n\n")
	response.WriteString(fmt.Sprintf("🤖 **Основной бот:**\n"))
	response.WriteString(fmt.Sprintf("  • Обработано задач: %d\n", ctx.Deps.WorkerPool.GetProcessedJobs()))
	response.WriteString(fmt.Sprintf("  • Неудачных задач: %d\n", ctx.Deps.WorkerPool.GetFailedJobs()))
	response.WriteString(fmt.Sprintf("  • Общее время обработки: %v\n", ctx.Deps.WorkerPool.GetProcessingTime()))
	response.WriteString(fmt.Sprintf("  • Размер очереди: %d\n\n", ctx.Deps.WorkerPool.GetQueueSize()))

	// Добавляем метрики command cache
	if ctx.Deps.CommandCache != nil {
		stats := ctx.Deps.CommandCache.Stats()
		response.WriteString(fmt.Sprintf("🗂️ **Command Cache:**\n"))
		response.WriteString(fmt.Sprintf("  • Размер кэша: %v\n", stats["size"]))
		response.WriteString(fmt.Sprintf("  • TTL: %v\n\n", stats["ttl"]))
	} else {
		response.WriteString("🗂️ **Command Cache:** Отключен\n\n")
	}

	response.WriteString("🔄 **Статус системы:**\n")
	response.WriteString("  • Все worker pool активны\n")
	response.WriteString("  • Система работает стабильно\n")

	return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, response.String())
}
