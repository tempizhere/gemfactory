package bot

import (
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"strings"
)

// parseArtists parses a comma-separated list of artists, handling spaces and special characters
func parseArtists(input string) []string {
	// Разделяем по запятым, учитывая пробелы
	rawArtists := strings.Split(input, ",")
	var artists []string
	for _, artist := range rawArtists {
		// Очищаем от пробелов
		cleaned := strings.TrimSpace(artist)
		if cleaned != "" {
			artists = append(artists, cleaned)
		}
	}
	return artists
}

// sendMessage sends a simple text message
func sendMessage(h *CommandHandlers, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := h.api.Send(msg); err != nil {
		h.logger.Error("Failed to send message", zap.Int64("chat_id", chatID), zap.String("text", text), zap.Error(err))
	}
}

// sendMessageWithMarkup sends a message with a reply markup
func sendMessageWithMarkup(h *CommandHandlers, chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = markup
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true // Отключаем превью ссылок
	if _, err := h.api.Send(msg); err != nil {
		h.logger.Error("Failed to send message with markup", zap.Int64("chat_id", chatID), zap.String("text", text), zap.Error(err))
	}
}
