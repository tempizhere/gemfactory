package admin

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/releases/cache"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap" // Добавлен импорт
	"strings"
)

// HandleAddArtist processes the /add_artist command
func HandleAddArtist(h *types.CommandHandlers, msg *tgbotapi.Message, args []string) {
	if len(args) < 2 {
		if err := h.API.SendMessage(msg.Chat.ID, "Использование: /add_artist <female|male> <artist1,artist2,...>"); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Использование: /add_artist <female|male> <artist1,artist2,...>"), zap.Error(err))
		}
		return
	}

	gender := strings.ToLower(args[0])
	isFemale := gender == "female"
	if gender != "female" && gender != "male" {
		if err := h.API.SendMessage(msg.Chat.ID, "Первый аргумент должен быть 'female' или 'male'. Пример: /add_artist female ITZY,aespa,IVE"); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Первый аргумент должен быть 'female' или 'male'. Пример: /add_artist female ITZY,aespa,IVE"), zap.Error(err))
		}
		return
	}

	artistsInput := strings.Join(args[1:], " ")
	artists := types.ParseArtists(artistsInput)
	if len(artists) == 0 {
		if err := h.API.SendMessage(msg.Chat.ID, "Не указаны артисты для добавления"); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Не указаны артисты для добавления"), zap.Error(err))
		}
		return
	}

	svc := service.NewArtistService(h.ArtistList, h.Logger)
	addedCount, err := svc.AddArtists(artists, isFemale)
	if err != nil {
		if err := h.API.SendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при добавлении артистов: %v", err)); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", fmt.Sprintf("Ошибка при добавлении артистов: %v", err)), zap.Error(err))
		}
		return
	}

	if addedCount == 0 {
		if err := h.API.SendMessage(msg.Chat.ID, "Ни один артист не добавлен, так как все указанные артисты уже в whitelist"); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Ни один артист не добавлен, так как все указанные артисты уже в whitelist"), zap.Error(err))
		}
		return
	}

	artistWord := "артист"
	if addedCount > 1 && addedCount < 5 {
		artistWord = "артиста"
	} else if addedCount >= 5 {
		artistWord = "артистов"
	}
	if err := h.API.SendMessage(msg.Chat.ID, fmt.Sprintf("Добавлено %d %s в %s whitelist", addedCount, artistWord, gender)); err != nil {
		h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", fmt.Sprintf("Добавлено %d %s в %s whitelist", addedCount, artistWord, gender)), zap.Error(err))
	}

	cache.ScheduleCacheUpdate(h.Config, h.Logger, h.ArtistList)
}

// HandleRemoveArtist processes the /remove_artist command
func HandleRemoveArtist(h *types.CommandHandlers, msg *tgbotapi.Message, args []string) {
	if len(args) < 1 {
		if err := h.API.SendMessage(msg.Chat.ID, "Использование: /remove_artist <artist1,artist2,...>"); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Использование: /remove_artist <artist1,artist2,...>"), zap.Error(err))
		}
		return
	}

	artistsInput := strings.Join(args, " ")
	artists := types.ParseArtists(artistsInput)
	if len(artists) == 0 {
		if err := h.API.SendMessage(msg.Chat.ID, "Не указаны артисты для удаления"); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Не указаны артисты для удаления"), zap.Error(err))
		}
		return
	}

	svc := service.NewArtistService(h.ArtistList, h.Logger)
	removedCount, err := svc.RemoveArtists(artists)
	if err != nil {
		if err := h.API.SendMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при удалении артистов: %v", err)); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", fmt.Sprintf("Ошибка при удалении артистов: %v", err)), zap.Error(err))
		}
		return
	}

	if removedCount == 0 {
		if err := h.API.SendMessage(msg.Chat.ID, "Ни один артист не удалён, так как указанные артисты отсутствуют в whitelist"); err != nil {
			h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", "Ни один артист не удалён, так как указанные артисты отсутствуют в whitelist"), zap.Error(err))
		}
		return
	}

	artistWord := "артист"
	if removedCount > 1 && removedCount < 5 {
		artistWord = "артиста"
	} else if removedCount >= 5 {
		artistWord = "артистов"
	}
	if err := h.API.SendMessage(msg.Chat.ID, fmt.Sprintf("Удалено %d %s из whitelist", removedCount, artistWord)); err != nil {
		h.Logger.Error("Failed to send message", zap.Int64("chat_id", msg.Chat.ID), zap.String("text", fmt.Sprintf("Удалено %d %s из whitelist", removedCount, artistWord)), zap.Error(err))
	}

	cache.ScheduleCacheUpdate(h.Config, h.Logger, h.ArtistList)
}
