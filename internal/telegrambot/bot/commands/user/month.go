package user

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/releasefmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

// HandleMonth processes the /month command
func HandleMonth(h *types.CommandHandlers, msg *tgbotapi.Message, args []string) {
	if len(args) == 0 {
		text := "Пожалуйста, выберите месяц:"
		reply := tgbotapi.NewMessage(msg.Chat.ID, text)
		reply.ReplyMarkup = h.Keyboard.GetMainKeyboard()
		types.SendMessageWithMarkup(h, msg.Chat.ID, text, reply.ReplyMarkup)
		return
	}

	month := strings.ToLower(args[0])
	femaleOnly := false
	maleOnly := false

	// Проверяем флаги -gg и -mg
	for _, arg := range args[1:] {
		if arg == "-gg" {
			femaleOnly = true
		} else if arg == "-mg" {
			maleOnly = true
		}
	}

	// Проверяем корректность месяца
	validMonth := false
	for _, m := range release.Months {
		if month == m {
			validMonth = true
			break
		}
	}
	if !validMonth {
		types.SendMessage(h, msg.Chat.ID, "Неверный месяц. Используйте /month [january, february, ...]")
		return
	}

	// Получаем вайтлист в зависимости от флагов
	var whitelist map[string]struct{}
	if femaleOnly && !maleOnly {
		whitelist = h.ArtistList.GetFemaleWhitelist()
	} else if maleOnly && !femaleOnly {
		whitelist = h.ArtistList.GetMaleWhitelist()
	} else {
		whitelist = h.ArtistList.GetUnitedWhitelist()
	}

	releases, err := cache.GetReleasesForMonths([]string{month}, whitelist, femaleOnly, maleOnly, h.ArtistList, h.Config, h.Logger)
	if err != nil {
		types.SendMessage(h, msg.Chat.ID, fmt.Sprintf("Ошибка при получении релизов: %v", err))
		return
	}

	if len(releases) == 0 {
		types.SendMessage(h, msg.Chat.ID, "Релизы не найдены.")
		return
	}

	var response strings.Builder
	for _, rel := range releases {
		formatted := releasefmt.FormatReleaseForTelegram(rel, h.Logger)
		response.WriteString(formatted + "\n")
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, response.String())
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = h.Keyboard.GetMainKeyboard()
	reply.DisableWebPagePreview = true
	types.SendMessageWithMarkup(h, msg.Chat.ID, response.String(), reply.ReplyMarkup)
}
