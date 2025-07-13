// Package service реализует бизнес-логику для работы с артистами и релизами.
package service

import (
	"fmt"
	"gemfactory/internal/telegrambot/releases/artist"
	"sort"
	"strings"

	"go.uber.org/zap"
)

// ArtistService handles business logic for artist whitelist operations
type ArtistService struct {
	artistList artist.WhitelistManager
	logger     *zap.Logger
}

// NewArtistService creates a new ArtistService instance
func NewArtistService(artistList artist.WhitelistManager, logger *zap.Logger) *ArtistService {
	return &ArtistService{
		artistList: artistList,
		logger:     logger,
	}
}

// ParseArtists parses a comma-separated list of artists
func (s *ArtistService) ParseArtists(input string) []string {
	rawArtists := strings.Split(input, ",")
	var artists []string
	for _, artist := range rawArtists {
		cleaned := strings.TrimSpace(artist)
		if cleaned != "" {
			artists = append(artists, cleaned)
		}
	}
	return artists
}

// FormatWhitelists formats the whitelists for display
func (s *ArtistService) FormatWhitelists() string {
	female := s.artistList.GetFemaleWhitelist()
	male := s.artistList.GetMaleWhitelist()

	var response strings.Builder

	response.WriteString("<b>Женские артисты:</b><code>\n")
	if len(female) == 0 {
		response.WriteString("пусто\n")
	} else {
		const columns = 3
		maxLength := 0
		for _, artist := range female {
			if len(artist) > maxLength {
				maxLength = len(artist)
			}
		}
		columnWidth := maxLength + 4
		rows := (len(female) + columns - 1) / columns
		for i := 0; i < rows; i++ {
			for j := 0; j < columns; j++ {
				index := i + j*rows
				if index < len(female) {
					response.WriteString(fmt.Sprintf("%-*s", columnWidth, female[index]))
				} else {
					response.WriteString(strings.Repeat(" ", columnWidth))
				}
			}
			response.WriteString("\n")
			if i > 0 && (i+1)%5 == 0 && i < rows-1 {
				response.WriteString("\n")
			}
		}
	}
	response.WriteString("</code>\n")

	response.WriteString("<b>Мужские артисты:</b><code>\n")
	if len(male) == 0 {
		response.WriteString("пусто\n")
	} else {
		const columns = 2
		maxLength := 0
		for _, artist := range male {
			if len(artist) > maxLength {
				maxLength = len(artist)
			}
		}
		columnWidth := maxLength + 4
		rows := (len(male) + columns - 1) / columns
		for i := 0; i < rows; i++ {
			for j := 0; j < columns; j++ {
				index := i + j*rows
				if index < len(male) {
					response.WriteString(fmt.Sprintf("%-*s", columnWidth, male[index]))
				} else {
					response.WriteString(strings.Repeat(" ", columnWidth))
				}
			}
			response.WriteString("\n")
			if i > 0 && (i+1)%5 == 0 && i < rows-1 {
				response.WriteString("\n")
			}
		}
	}
	response.WriteString("</code>\n")

	return response.String()
}

// FormatWhitelistsForExport formats the whitelists for export
func (s *ArtistService) FormatWhitelistsForExport() string {
	female := s.artistList.GetFemaleWhitelist()
	male := s.artistList.GetMaleWhitelist()

	var response strings.Builder

	response.WriteString("<b>Женские артисты:</b>\n")
	if len(female) == 0 {
		response.WriteString("пусто\n")
	} else {
		sort.Strings(female)
		response.WriteString(fmt.Sprintf("<code>%s</code>\n", strings.Join(female, ", ")))
	}

	response.WriteString("<b>Мужские артисты:</b>\n")
	if len(male) == 0 {
		response.WriteString("пусто\n")
	} else {
		sort.Strings(male)
		response.WriteString(fmt.Sprintf("<code>%s</code>\n", strings.Join(male, ", ")))
	}

	return response.String()
}

// AddArtists adds artists to the specified whitelist
func (s *ArtistService) AddArtists(artists []string, isFemale bool) (int, error) {
	count, err := s.artistList.AddArtists(artists, isFemale)
	if err != nil {
		s.logger.Error("Failed to add artists", zap.Strings("artists", artists), zap.Error(err))
		return count, fmt.Errorf("failed to add artists: %w", err)
	}
	return count, nil
}

// RemoveArtists removes artists from the whitelist
func (s *ArtistService) RemoveArtists(artists []string) (int, error) {
	count, err := s.artistList.RemoveArtists(artists)
	if err != nil {
		s.logger.Error("Failed to remove artists", zap.Strings("artists", artists), zap.Error(err))
		return count, fmt.Errorf("failed to remove artists: %w", err)
	}
	return count, nil
}

// ClearWhitelists clears both whitelists
func (s *ArtistService) ClearWhitelists() error {
	err := s.artistList.ClearWhitelists()
	if err != nil {
		s.logger.Error("Failed to clear whitelists", zap.Error(err))
		return fmt.Errorf("failed to clear whitelists: %w", err)
	}
	return nil
}

// GetFemaleWhitelist returns the female whitelist
func (s *ArtistService) GetFemaleWhitelist() []string {
	return s.artistList.GetFemaleWhitelist()
}

// GetMaleWhitelist returns the male whitelist
func (s *ArtistService) GetMaleWhitelist() []string {
	return s.artistList.GetMaleWhitelist()
}
