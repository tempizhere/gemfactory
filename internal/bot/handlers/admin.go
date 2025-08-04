// Package commands содержит обработчики команд для Telegram-бота.
package commands

import (
	"context"
	"fmt"
	"gemfactory/internal/bot/middleware"
	"gemfactory/internal/bot/router"
	"gemfactory/internal/domain/types"
	"strings"
	"time"
)

// RegisterAdminRoutes registers admin command handlers
func RegisterAdminRoutes(r *router.Router, deps *types.Dependencies) {
	r.Handle("add_artist", middleware.Wrap(middleware.AdminOnly(deps.Config.GetAdminUsername()), handleAddArtist))
	r.Handle("remove_artist", middleware.Wrap(middleware.AdminOnly(deps.Config.GetAdminUsername()), handleRemoveArtist))
	r.Handle("clearcache", middleware.Wrap(middleware.AdminOnly(deps.Config.GetAdminUsername()), handleClearCache))
	r.Handle("clearwhitelists", middleware.Wrap(middleware.AdminOnly(deps.Config.GetAdminUsername()), handleClearWhitelists))
	r.Handle("export", middleware.Wrap(middleware.AdminOnly(deps.Config.GetAdminUsername()), handleExport))

}

func handleAddArtist(ctx types.Context) error {
	args := strings.Fields(ctx.Message.Text)[1:]
	if len(args) < 2 {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Использование: /add_artist <female|male> <artist1,artist2,...>")
	}

	gender := strings.ToLower(args[0])
	isFemale := gender == "female"
	if gender != "female" && gender != "male" {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Первый аргумент должен быть 'female' или 'male'. Пример: /add_artist female ITZY,aespa")
	}

	artistsInput := strings.Join(args[1:], " ")
	artists := ctx.Deps.ArtistService.ParseArtists(artistsInput)
	if len(artists) == 0 {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Не указаны артисты для добавления")
	}

	addedCount, err := ctx.Deps.ArtistService.AddArtists(artists, isFemale)
	if err != nil {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, fmt.Sprintf("Ошибка при добавлении артистов: %v", err))
	}

	if addedCount == 0 {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Ни один артист не добавлен, так как все уже в whitelist")
	}

	artistWord := "артист"
	if addedCount > 1 && addedCount < 5 {
		artistWord = "артиста"
	} else if addedCount >= 5 {
		artistWord = "артистов"
	}
	updateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	ctx.Deps.Cache.ScheduleUpdate(updateCtx)
	return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, fmt.Sprintf("Добавлено %d %s в %s whitelist", addedCount, artistWord, gender))
}

func handleRemoveArtist(ctx types.Context) error {
	args := strings.Fields(ctx.Message.Text)[1:]
	if len(args) < 1 {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Использование: /remove_artist <artist1,artist2,...>")
	}

	artistsInput := strings.Join(args, " ")
	artists := ctx.Deps.ArtistService.ParseArtists(artistsInput)
	if len(artists) == 0 {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Не указаны артисты для удаления")
	}

	removedCount, err := ctx.Deps.ArtistService.RemoveArtists(artists)
	if err != nil {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, fmt.Sprintf("Ошибка при удалении артистов: %v", err))
	}

	if removedCount == 0 {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Ни один артист не удалён, так как указанные артисты отсутствуют")
	}

	artistWord := "артист"
	if removedCount > 1 && removedCount < 5 {
		artistWord = "артиста"
	} else if removedCount >= 5 {
		artistWord = "артистов"
	}
	updateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	ctx.Deps.Cache.ScheduleUpdate(updateCtx)
	return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, fmt.Sprintf("Удалено %d %s из whitelist", removedCount, artistWord))
}

func handleClearCache(ctx types.Context) error {
	ctx.Deps.ReleaseService.ClearCache()

	return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Кэш очищен, обновление запущено")
}

func handleClearWhitelists(ctx types.Context) error {
	if err := ctx.Deps.ArtistService.ClearWhitelists(); err != nil {
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, fmt.Sprintf("Ошибка при очистке вайтлистов: %v", err))
	}
	return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Вайтлисты очищены")
}

func handleExport(ctx types.Context) error {
	response := ctx.Deps.ArtistService.FormatWhitelistsForExport()
	return ctx.Deps.BotAPI.SendMessageWithMarkup(ctx.Message.Chat.ID, response, ctx.Deps.Keyboard.GetMainKeyboard())
}
