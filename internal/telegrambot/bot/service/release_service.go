package service

import (
	"fmt"
	"gemfactory/internal/telegrambot/releases/artist"
	"gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/release"
	"gemfactory/internal/telegrambot/releases/service"
	"gemfactory/pkg/config"
	"strings"

	"go.uber.org/zap"
)

// ReleaseService handles business logic for release-related operations
type ReleaseService struct {
	artistList artist.WhitelistManager
	config     *config.Config
	logger     *zap.Logger
	cache      cache.Cache
}

// NewReleaseService creates a new ReleaseService instance
func NewReleaseService(artistList artist.WhitelistManager, config *config.Config, logger *zap.Logger, cache cache.Cache) *ReleaseService {
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
	cfg := release.NewConfig()
	validMonth := false
	for _, m := range cfg.Months() {
		if month == m {
			validMonth = true
			break
		}
	}
	if !validMonth {
		return "", fmt.Errorf("invalid month: %s", month)
	}

	whitelistSlice := s.artistList.GetUnitedWhitelist()
	if len(whitelistSlice) == 0 {
		return "", fmt.Errorf("whitelist is empty, please add artists")
	}

	whitelist := make(map[string]struct{})
	for _, artist := range whitelistSlice {
		whitelist[artist] = struct{}{}
	}

	var targetWhitelist map[string]struct{}
	switch {
	case femaleOnly && !maleOnly:
		femaleSlice := s.artistList.GetFemaleWhitelist()
		targetWhitelist = make(map[string]struct{})
		for _, artist := range femaleSlice {
			targetWhitelist[artist] = struct{}{}
		}
	case !femaleOnly && maleOnly:
		maleSlice := s.artistList.GetMaleWhitelist()
		targetWhitelist = make(map[string]struct{})
		for _, artist := range maleSlice {
			targetWhitelist[artist] = struct{}{}
		}
	default:
		targetWhitelist = whitelist
	}

	releases, missingMonths, err := s.cache.GetReleasesForMonths([]string{month}, targetWhitelist, femaleOnly, maleOnly)
	if err != nil {
		s.logger.Error("Failed to get releases", zap.String("month", month), zap.Error(err))
		return "", fmt.Errorf("failed to get releases: %w", err)
	}

	if len(releases) > 0 {
		var response strings.Builder
		for _, rel := range releases {
			formatted := service.FormatReleaseForTelegram(rel)
			response.WriteString(formatted + "\n")
		}
		s.logger.Debug("Returning releases", zap.String("month", month), zap.Int("response_length", len(response.String())))
		return strings.TrimSpace(response.String()), nil
	}

	for _, missing := range missingMonths {
		if strings.ToLower(missing) == month {
			if s.cache.IsUpdating(month) {
				return fmt.Sprintf("Данные для %s обновляются. Попробуйте снова через минуту.", month), nil
			}
			return fmt.Sprintf("Релизы для %s еще не анонсированы.", month), nil
		}
	}

	return fmt.Sprintf("Релизы для %s не найдены.", month), nil
}

// ClearCache clears the release cache
func (s *ReleaseService) ClearCache() {
	s.cache.Clear()
	go s.cache.ScheduleUpdate()
}
