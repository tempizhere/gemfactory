package service

import (
	"errors"
	"fmt"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/releasefmt"
	"gemfactory/pkg/config"
	"go.uber.org/zap"
	"strings"
)

// ReleaseService handles business logic for release-related operations
type ReleaseService struct {
	artistList *artistlist.ArtistList
	config     *config.Config
	logger     *zap.Logger
	cache      cache.Cache
}

// NewReleaseService creates a new ReleaseService instance
func NewReleaseService(artistList *artistlist.ArtistList, config *config.Config, logger *zap.Logger, cache cache.Cache) *ReleaseService {
	return &ReleaseService{
		artistList: artistList,
		config:     config,
		logger:     logger,
		cache:      cache,
	}
}

// GetReleasesForMonth retrieves and formats releases for a given month
func (s *ReleaseService) GetReleasesForMonth(month string, femaleOnly, maleOnly bool) (string, error) {
	month = strings.ToLower(month)
	validMonth := false
	for _, m := range release.Months {
		if month == m {
			validMonth = true
			break
		}
	}
	if !validMonth {
		return "", fmt.Errorf("invalid month: %s", month)
	}

	if len(s.artistList.GetUnitedWhitelist()) == 0 {
		return "", fmt.Errorf("whitelist is empty, please add artists")
	}

	var whitelist map[string]struct{}
	if femaleOnly && !maleOnly {
		whitelist = s.artistList.GetFemaleWhitelist()
	} else if maleOnly && !femaleOnly {
		whitelist = s.artistList.GetMaleWhitelist()
	} else {
		whitelist = s.artistList.GetUnitedWhitelist()
	}

	releases, err := s.cache.GetReleasesForMonths([]string{month}, whitelist, femaleOnly, maleOnly)
	if err != nil {
		if errors.Is(err, cache.ErrNoCache) {
			return "Релизы для этого месяца пока недоступны. Попробуйте позже!", nil
		}
		return "", fmt.Errorf("failed to get releases: %v", err)
	}

	if len(releases) == 0 {
		return "Релизы не найдены.", nil
	}

	var response strings.Builder
	for _, rel := range releases {
		formatted := releasefmt.FormatReleaseForTelegram(rel, s.logger)
		response.WriteString(formatted + "\n")
	}
	return response.String(), nil
}

// ClearCache clears the release cache and reinitializes it
func (s *ReleaseService) ClearCache() {
	s.cache.Clear()
	go s.cache.ScheduleUpdate()
}
