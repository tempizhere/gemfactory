package service

import (
	"fmt"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"go.uber.org/zap"
	"sort"
	"strings"
)

// ArtistService handles business logic for artist whitelist operations
type ArtistService struct {
	artistList *artistlist.ArtistList
	logger     *zap.Logger
}

// NewArtistService creates a new ArtistService instance
func NewArtistService(artistList *artistlist.ArtistList, logger *zap.Logger) *ArtistService {
	return &ArtistService{
		artistList: artistList,
		logger:     logger,
	}
}

// FormatWhitelists formats the female and male whitelists for display
func (s *ArtistService) FormatWhitelists() string {
	female := s.artistList.GetFemaleWhitelist()
	male := s.artistList.GetMaleWhitelist()

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
		maxLength := 0
		for _, artist := range femaleArtists {
			if len(artist) > maxLength {
				maxLength = len(artist)
			}
		}
		columnWidth := maxLength + 4
		rows := (len(femaleArtists) + columns - 1) / columns
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
		maxLength := 0
		for _, artist := range maleArtists {
			if len(artist) > maxLength {
				maxLength = len(artist)
			}
		}
		columnWidth := maxLength + 4
		rows := (len(maleArtists) + columns - 1) / columns
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
			if i > 0 && (i+1)%5 == 0 && i < rows-1 {
				response.WriteString("\n")
			}
		}
	}
	response.WriteString("</code>\n")

	return response.String()
}

// FormatWhitelistsForExport formats the whitelists in a compact format for export
func (s *ArtistService) FormatWhitelistsForExport() string {
	female := s.artistList.GetFemaleWhitelist()
	male := s.artistList.GetMaleWhitelist()

	var response strings.Builder

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

	return response.String()
}

// AddArtists adds artists to the specified whitelist
func (s *ArtistService) AddArtists(artists []string, isFemale bool) (int, error) {
	return s.artistList.AddArtists(artists, isFemale)
}

// RemoveArtists removes artists from the whitelist
func (s *ArtistService) RemoveArtists(artists []string) (int, error) {
	return s.artistList.RemoveArtists(artists)
}

// ClearWhitelists clears both female and male whitelists
func (s *ArtistService) ClearWhitelists() error {
	return s.artistList.ClearWhitelists()
}
