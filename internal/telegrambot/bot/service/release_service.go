package service

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/releasefmt"
	"gemfactory/pkg/config"
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

	releases, missingMonths, err := s.cache.GetReleasesForMonths([]string{month}, whitelist, femaleOnly, maleOnly)
	if err != nil {
		return "", fmt.Errorf("failed to get releases: %v", err)
	}

	// Проверяем, спарсен ли месяц (есть ли ссылки в кэше)
	links, err := s.cache.GetCachedLinks(month)
	if err != nil || len(links) == 0 {
		return "Релизы для этого месяца еще не анонсированы.", nil
	}

	// Если месяц отсутствует в кэше
	if len(missingMonths) > 0 {
		if s.cache.IsUpdating(month) {
			return fmt.Sprintf("Данные для %s обновляются. Попробуйте снова через минуту.", month), nil
		}
		// Если кэш не обновляется, но ссылки есть, значит релизов нет
		return fmt.Sprintf("Релизы для %s не найдены.", month), nil
	}

	if len(releases) == 0 {
		return fmt.Sprintf("Релизы для %s не найдены.", month), nil
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
