package bot

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sort"
	"strings"
)

// handleWhitelists processes the /whitelists command
func handleWhitelists(h *CommandHandlers, msg *tgbotapi.Message) {
	female := h.al.GetFemaleWhitelist()
	male := h.al.GetMaleWhitelist()

	var response strings.Builder
	response.WriteString("<b>Женские артисты:</b><code>\n")
	femaleArtists := make([]string, 0, len(female))
	for artist := range female {
		femaleArtists = append(femaleArtists, artist)
	}
	sort.Strings(femaleArtists)
	if len(femaleArtists) == 0 {
		response.WriteString("пусто\n")
	} else {
		const columns = 3
		// Находим максимальную длину имени
		maxLength := 0
		for _, artist := range femaleArtists {
			if len(artist) > maxLength {
				maxLength = len(artist)
			}
		}
		columnWidth := maxLength + 4 // Увеличенный отступ
		// Рассчитываем количество строк
		rows := (len(femaleArtists) + columns - 1) / columns
		// Заполняем столбцы вертикально
		for i := 0; i < rows; i++ {
			for j := 0; j < columns; j++ {
				index := i + j*rows
				if index < len(femaleArtists) {
					response.WriteString(fmt.Sprintf("%-*s", columnWidth, femaleArtists[index]))
				} else {
					response.WriteString(strings.Repeat(" ", columnWidth))
				}
			}
			response.WriteString("\n")
			// Добавляем пустую строку каждые 5 строк
			if i > 0 && (i+1)%5 == 0 && i < rows-1 {
				response.WriteString("\n")
			}
		}
	}
	response.WriteString("</code>\n")

	response.WriteString("<b>Мужские артисты:</b><code>\n")
	maleArtists := make([]string, 0, len(male))
	for artist := range male {
		maleArtists = append(maleArtists, artist)
	}
	sort.Strings(maleArtists)
	if len(maleArtists) == 0 {
		response.WriteString("пусто\n")
	} else {
		const columns = 2
		// Находим максимальную длину имени
		maxLength := 0
		for _, artist := range maleArtists {
			if len(artist) > maxLength {
				maxLength = len(artist)
			}
		}
		columnWidth := maxLength + 4 // Увеличенный отступ
		// Рассчитываем количество строк
		rows := (len(maleArtists) + columns - 1) / columns
		// Заполняем столбцы вертикально
		for i := 0; i < rows; i++ {
			for j := 0; j < columns; j++ {
				index := i + j*rows
				if index < len(maleArtists) {
					response.WriteString(fmt.Sprintf("%-*s", columnWidth, maleArtists[index]))
				} else {
					response.WriteString(strings.Repeat(" ", columnWidth))
				}
			}
			response.WriteString("\n")
			// Добавляем пустую строку каждые 5 строк
			if i > 0 && (i+1)%5 == 0 && i < rows-1 {
				response.WriteString("\n")
			}
		}
	}
	response.WriteString("</code>\n")

	reply := tgbotapi.NewMessage(msg.Chat.ID, response.String())
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = h.keyboard.GetMainKeyboard()
	sendMessageWithMarkup(h, msg.Chat.ID, response.String(), reply.ReplyMarkup)
}
