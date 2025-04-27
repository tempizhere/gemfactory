package admin

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/releases/cache"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

// HandleAddArtist processes the /add_artist command
func HandleAddArtist(h *types.CommandHandlers, msg *tgbotapi.Message, args []string) {
	if len(args) < 2 {
		types.SendMessage(h, msg.Chat.ID, "Использование: /add_artist <female|male> <artist1,artist2,...>")
		return
	}

	gender := strings.ToLower(args[0])
	isFemale := gender == "female"
	if gender != "female" && gender != "male" {
		types.SendMessage(h, msg.Chat.ID, "Первый аргумент должен быть 'female' или 'male'. Пример: /add_artist female ITZY,aespa,IVE")
		return
	}

	// Объединяем аргументы, начиная со второго, и парсим список артистов
	artistsInput := strings.Join(args[1:], " ")
	artists := types.ParseArtists(artistsInput)
	if len(artists) == 0 {
		types.SendMessage(h, msg.Chat.ID, "Не указаны артисты для добавления")
		return
	}

	addedCount, err := h.ArtistList.AddArtists(artists, isFemale)
	if err != nil {
		types.SendMessage(h, msg.Chat.ID, fmt.Sprintf("Ошибка при добавлении артистов: %v", err))
		return
	}

	if addedCount == 0 {
		types.SendMessage(h, msg.Chat.ID, "Ни один артист не добавлен, так как все указанные артисты уже в whitelist")
		return
	}

	artistWord := "артист"
	if addedCount > 1 && addedCount < 5 {
		artistWord = "артиста"
	} else if addedCount >= 5 {
		artistWord = "артистов"
	}
	types.SendMessage(h, msg.Chat.ID, fmt.Sprintf("Добавлено %d %s в %s whitelist", addedCount, artistWord, gender))

	// Запускаем обновление кэша асинхронно
	go cache.InitializeCache(h.Config, h.Logger, h.ArtistList)
}

// HandleRemoveArtist processes the /remove_artist command
func HandleRemoveArtist(h *types.CommandHandlers, msg *tgbotapi.Message, args []string) {
	if len(args) < 1 {
		types.SendMessage(h, msg.Chat.ID, "Использование: /remove_artist <artist1,artist2,...>")
		return
	}

	// Объединяем аргументы и парсим список артистов
	artistsInput := strings.Join(args, " ")
	artists := types.ParseArtists(artistsInput)
	if len(artists) == 0 {
		types.SendMessage(h, msg.Chat.ID, "Не указаны артисты для удаления")
		return
	}

	removedCount, err := h.ArtistList.RemoveArtists(artists)
	if err != nil {
		types.SendMessage(h, msg.Chat.ID, fmt.Sprintf("Ошибка при удаления артистов: %v", err))
		return
	}

	if removedCount == 0 {
		types.SendMessage(h, msg.Chat.ID, "Ни один артист не удалён, так как указанные артисты отсутствуют в whitelist")
		return
	}

	artistWord := "артист"
	if removedCount > 1 && removedCount < 5 {
		artistWord = "артиста"
	} else if removedCount >= 5 {
		artistWord = "артистов"
	}
	types.SendMessage(h, msg.Chat.ID, fmt.Sprintf("Удалено %d %s из whitelist", removedCount, artistWord))

	// Запускаем обновление кэша асинхронно
	go cache.InitializeCache(h.Config, h.Logger, h.ArtistList)
}
