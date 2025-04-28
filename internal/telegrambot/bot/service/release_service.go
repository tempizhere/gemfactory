package service

import (
	"fmt"
	"gemfactory/internal/telegrambot/releases/artistlist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/releasefmt"
	"gemfactory/pkg/config"
	"go.uber.org/zap"
	"strings"
	"time"
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

	// Проверяем, является ли месяц будущим
	currentTime := time.Now()
	currentMonth := strings.ToLower(currentTime.Month().String())
	monthOrder := map[string]int{
		"january":   1,
		"february":  2,
		"march":     3,
		"april":     4,
		"may":       5,
		"june":      6,
		"july":      7,
		"august":    8,
		"september": 9,
		"october":   10,
		"november":  11,
		"december":  12,
	}
	currentMonthNum := monthOrder[currentMonth]
	requestedMonthNum := monthOrder[month]
	if requestedMonthNum > currentMonthNum {
		return "Релизы для этого месяца еще не анонсированы.", nil
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
