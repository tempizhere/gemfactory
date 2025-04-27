package bot

import (
	"fmt"
	"gemfactory/internal/features/releasesbot/cache"
	"gemfactory/internal/features/releasesbot/release"
	"gemfactory/internal/features/releasesbot/releasefmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

// handleMonth processes the /month command
func handleMonth(h *CommandHandlers, msg *tgbotapi.Message, args []string) {
	if len(args) == 0 {
		text := "Пожалуйста, выберите месяц:"
		reply := tgbotapi.NewMessage(msg.Chat.ID, text)
		reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
		sendMessageWithMarkup(h, msg.Chat.ID, text, reply.ReplyMarkup)
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
		sendMessage(h, msg.Chat.ID, "Неверный месяц. Используйте /month [january, february, ...]")
		return
	}

	// Получаем вайтлист в зависимости от флагов
	var whitelist map[string]struct{}
	if femaleOnly && !maleOnly {
		whitelist = h.al.GetFemaleWhitelist()
	} else if maleOnly && !femaleOnly {
		whitelist = h.al.GetMaleWhitelist()
	} else {
		whitelist = h.al.GetUnitedWhitelist()
	}

	releases, err := cache.GetReleasesForMonths([]string{month}, whitelist, femaleOnly, maleOnly, h.al, h.config, h.logger)
	if err != nil {
		sendMessage(h, msg.Chat.ID, fmt.Sprintf("Ошибка при получении релизов: %v", err))
		return
	}

	if len(releases) == 0 {
		sendMessage(h, msg.Chat.ID, "Релизы не найдены.")
		return
	}

	var response strings.Builder
	for _, rel := range releases {
		formatted := releasefmt.FormatReleaseForTelegram(rel, h.logger)
		response.WriteString(formatted + "\n")
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, response.String())
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
	reply.DisableWebPagePreview = true
	sendMessageWithMarkup(h, msg.Chat.ID, response.String(), reply.ReplyMarkup)
}
