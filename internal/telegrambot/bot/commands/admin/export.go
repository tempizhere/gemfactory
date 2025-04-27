package admin

import (
	"fmt"
	"gemfactory/internal/telegrambot/bot/types"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sort"
	"strings"
)

// HandleExport processes the /export command
func HandleExport(h *types.CommandHandlers, msg *tgbotapi.Message) {
	female := h.ArtistList.GetFemaleWhitelist()
	male := h.ArtistList.GetMaleWhitelist()

	var response strings.Builder

	// Формируем список женских артистов
	response.WriteString("<b>Женские артисты:</b>\n")
	femaleArtists := make([]string, 0, len(female))
	for artist := range female {
		femaleArtists = append(femaleArtists, artist)
	}
	sort.Strings(femaleArtists)
	if len(femaleArtists) == 0 {
		response.WriteString("пусто\n")
	} else {
		response.WriteString(fmt.Sprintf("<code>%s</code>\n", strings.Join(femaleArtists, ", ")))
	}

	// Формируем список мужских артистов
	response.WriteString("<b>Мужские артисты:</b>\n")
	maleArtists := make([]string, 0, len(male))
	for artist := range male {
		maleArtists = append(maleArtists, artist)
	}
	sort.Strings(maleArtists)
	if len(maleArtists) == 0 {
		response.WriteString("пусто\n")
	} else {
		response.WriteString(fmt.Sprintf("<code>%s</code>\n", strings.Join(maleArtists, ", ")))
	}

	types.SendMessageWithMarkup(h, msg.Chat.ID, response.String(), h.Keyboard.GetMainKeyboard())
}
