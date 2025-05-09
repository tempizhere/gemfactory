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
		"\n" +
		fmt.Sprintf("По вопросам вайтлистов: @%s", ctx.Deps.Config.AdminUsername)
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
