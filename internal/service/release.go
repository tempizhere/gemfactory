// Package service содержит бизнес-логику приложения.
package service

import (
	"context"
	"fmt"
	"gemfactory/internal/external/scraper"
	"gemfactory/internal/model"
	"gemfactory/internal/storage/repository"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ReleaseService содержит бизнес-логику для работы с релизами
type ReleaseService struct {
	repo            model.ReleaseRepository
	releaseTypeRepo model.ReleaseTypeRepository
	artistRepo      model.ArtistRepository
	scraper         scraper.Fetcher
	logger          *zap.Logger
	utils           *model.ReleaseUtils
}

// NewReleaseService создает новый сервис релизов
func NewReleaseService(db *bun.DB, scraper scraper.Fetcher, logger *zap.Logger) *ReleaseService {
	return &ReleaseService{
		repo:            repository.NewReleaseRepository(db, logger),
		releaseTypeRepo: repository.NewReleaseTypeRepository(db, logger),
		artistRepo:      repository.NewArtistRepository(db, logger),
		scraper:         scraper,
		logger:          logger,
		utils:           model.NewReleaseUtils(),
	}
}

// GetReleasesForMonth возвращает релизы за месяц с фильтром
func (s *ReleaseService) GetReleasesForMonth(month string, femaleOnly, maleOnly bool) (string, error) {
	// Нормализуем месяц
	month = strings.ToLower(month)

	// Парсим месяц и год из строки (например, "september-2024")
	var year int
	if strings.Contains(month, "-") {
		parts := strings.Split(month, "-")
		if len(parts) == 2 {
			month = parts[0]
			if parsedYear, err := strconv.Atoi(parts[1]); err == nil {
				year = parsedYear
			}
		}
	}

	// Определяем пол по фильтру
	var gender string
	if femaleOnly {
		gender = "female"
	} else if maleOnly {
		gender = "male"
	}

	// Получаем релизы
	var releases []model.Release
	var err error

	if gender != "" {
		var genderType model.Gender
		switch gender {
		case "female":
			genderType = model.GenderFemale
		case "male":
			genderType = model.GenderMale
		default:
			genderType = model.GenderMixed
		}

		if year > 0 {
			releases, err = s.repo.GetByMonthYearAndGender(month, year, genderType)
		} else {
			releases, err = s.repo.GetByMonthAndGender(month, genderType)
		}
	} else {
		if year > 0 {
			releases, err = s.repo.GetByMonthAndYear(month, year)
		} else {
			releases, err = s.repo.GetByMonth(month)
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to get releases for month %s: %w", month, err)
	}

	// Форматируем ответ
	var result strings.Builder
	result.WriteString(fmt.Sprintf("🎵 Релизы за %s:\n\n", month))

	if len(releases) == 0 {
		result.WriteString("Релизы не найдены")
		return result.String(), nil
	}

	for _, release := range releases {
		var artistName string
		if release.Artist != nil {
			artistName = release.Artist.Name
		}

		// Форматируем релиз в нужном формате: дата | <b>АРТИСТ</b> | название (ссылка)
		line := fmt.Sprintf("%s | <b>%s</b>", release.Date, html.EscapeString(artistName))

		// Добавляем название релиза
		if release.Title != "" && release.Title != "N/A" {
			line += fmt.Sprintf(" | %s", html.EscapeString(release.Title))
		}

		// Добавляем ссылку на MV если есть (встраиваем в TitleTrack)
		if release.MV != "" && release.MV != "N/A" {
			// Очищаем TitleTrack
			cleanedTitleTrack := strings.ReplaceAll(release.TitleTrack, "Title Track:", "")
			cleanedTitleTrack = strings.TrimSpace(cleanedTitleTrack)

			if cleanedTitleTrack != "" && cleanedTitleTrack != "N/A" {
				// Встраиваем ссылку в название трека
				trackName := html.EscapeString(cleanedTitleTrack)
				line += fmt.Sprintf(" | <a href=\"%s\">%s</a>", release.MV, trackName)
			} else {
				// Если нет названия трека, добавляем просто ссылку
				line += fmt.Sprintf(" | <a href=\"%s\">Link</a>", release.MV)
			}
		} else if release.TitleTrack != "" && release.TitleTrack != "N/A" {
			// Если нет ссылки, но есть название трека, добавляем его
			cleanedTitleTrack := strings.ReplaceAll(release.TitleTrack, "Title Track:", "")
			cleanedTitleTrack = strings.TrimSpace(cleanedTitleTrack)
			line += fmt.Sprintf(" | %s", html.EscapeString(cleanedTitleTrack))
		}

		result.WriteString(line + "\n")
	}

	return result.String(), nil
}

// CreateOrUpdateRelease создает новый релиз или обновляет существующий
func (s *ReleaseService) CreateOrUpdateRelease(release *model.Release) error {
	// Валидируем релиз
	if err := s.utils.ValidateRelease(release); err != nil {
		return fmt.Errorf("release validation failed: %w", err)
	}

	// Очищаем данные
	release.Title = s.utils.CleanReleaseTitle(release.Title)
	release.AlbumName = s.utils.CleanReleaseTitle(release.AlbumName)
	release.TitleTrack = s.utils.CleanReleaseTitle(release.TitleTrack)

	// Определяем тип релиза, если не указан
	if release.ReleaseTypeID == 0 {
		// Получаем тип релиза по умолчанию (single)
		releaseType, err := s.releaseTypeRepo.GetByName("single")
		if err != nil {
			return fmt.Errorf("failed to get default release type: %w", err)
		}
		if releaseType != nil {
			release.ReleaseTypeID = releaseType.ReleaseTypeID
		}
	}

	// Парсим дату для извлечения месяца и года
	if release.Date != "" {
		if parsedDate, err := s.utils.ParseReleaseDate(release.Date); err == nil {
			release.Month = s.utils.ExtractMonthFromDate(parsedDate)
			release.Year = s.utils.ExtractYearFromDate(parsedDate)
		}
	}

	// Проверяем, существует ли релиз по артисту и дате
	existingRelease, err := s.repo.GetByArtistAndDate(release.ArtistID, release.Date)
	if err != nil {
		return fmt.Errorf("failed to check for existing release: %w", err)
	}

	if existingRelease != nil {
		// Релиз существует, обновляем его
		s.logger.Info("Release exists, updating",
			zap.String("artist_id", fmt.Sprintf("%d", release.ArtistID)),
			zap.String("date", release.Date),
			zap.String("old_track", existingRelease.TitleTrack),
			zap.String("new_track", release.TitleTrack),
			zap.String("old_youtube", existingRelease.MV),
			zap.String("new_youtube", release.MV))

		// Обновляем поля существующего релиза
		existingRelease.AlbumName = release.AlbumName
		existingRelease.TitleTrack = release.TitleTrack
		existingRelease.MV = release.MV
		existingRelease.TimeMSK = release.TimeMSK
		existingRelease.ReleaseTypeID = release.ReleaseTypeID
		existingRelease.UpdatedAt = time.Now()

		s.logger.Info("Updated release fields",
			zap.String("old_album", existingRelease.AlbumName),
			zap.String("new_album", release.AlbumName),
			zap.String("old_track", existingRelease.TitleTrack),
			zap.String("new_track", release.TitleTrack),
			zap.String("old_youtube", existingRelease.MV),
			zap.String("new_youtube", release.MV))

		return s.repo.Update(existingRelease)
	} else {
		// Релиз не существует, создаем новый
		s.logger.Info("Release not found, creating new",
			zap.String("artist_id", fmt.Sprintf("%d", release.ArtistID)),
			zap.String("date", release.Date),
			zap.String("track", release.TitleTrack),
			zap.String("youtube", release.MV))

		return s.repo.Create(release)
	}
}

// UpdateRelease обновляет релиз
func (s *ReleaseService) UpdateRelease(release *model.Release) error {
	// Валидируем релиз
	if err := s.utils.ValidateRelease(release); err != nil {
		return fmt.Errorf("release validation failed: %w", err)
	}

	// Очищаем данные
	release.Title = s.utils.CleanReleaseTitle(release.Title)
	release.AlbumName = s.utils.CleanReleaseTitle(release.AlbumName)
	release.TitleTrack = s.utils.CleanReleaseTitle(release.TitleTrack)

	return s.repo.Update(release)
}

// FormatReleaseForDisplay форматирует релиз для отображения
func (s *ReleaseService) FormatReleaseForDisplay(release *model.Release) string {
	return s.utils.FormatReleaseForDisplay(release)
}

// GetReleaseConfig возвращает конфигурацию релизов
func (s *ReleaseService) GetReleaseConfig() *model.ReleaseConfig {
	return model.NewReleaseConfig()
}

// FormatDate форматирует дату с кэшированием
func (s *ReleaseService) FormatDate(dateStr string) (string, error) {
	parsedDate, err := s.utils.ParseReleaseDate(dateStr)
	if err != nil {
		return "", err
	}
	return s.utils.FormatReleaseDate(parsedDate), nil
}

// FormatTimeKST форматирует время KST
func (s *ReleaseService) FormatTimeKST(timeStr string) (string, error) {
	parsedTime, err := s.utils.ParseReleaseTime(timeStr)
	if err != nil {
		return "", err
	}
	return s.utils.FormatReleaseTime(parsedTime), nil
}

// ConvertKSTToMSK конвертирует время из KST в MSK
func (s *ReleaseService) ConvertKSTToMSK(kstTimeStr string) (string, error) {
	return s.utils.ConvertKSTToMSKString(kstTimeStr)
}

// CleanLink очищает ссылку
func (s *ReleaseService) CleanLink(link string) string {
	return s.utils.CleanLink(link)
}

// FormatReleaseForTelegram форматирует релиз для Telegram
func (s *ReleaseService) FormatReleaseForTelegram(release *model.Release) string {
	return s.utils.FormatReleaseForTelegram(release)
}

// AddRelease добавляет новый релиз
func (s *ReleaseService) AddRelease(release *model.Release) error {
	// Проверяем, существует ли уже такой релиз
	existing, err := s.repo.GetByArtist(release.ArtistID)
	if err != nil {
		return fmt.Errorf("failed to check existing releases: %w", err)
	}

	// Проверяем дубликаты
	for _, existingRelease := range existing {
		if existingRelease.Title == release.Title && existingRelease.Date == release.Date {
			var artistName string
			if release.Artist != nil {
				artistName = release.Artist.Name
			}
			return fmt.Errorf("release already exists: %s - %s", artistName, release.Title)
		}
	}

	// Создаем релиз
	err = s.repo.Create(release)
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	return nil
}

// DeleteRelease удаляет релиз
func (s *ReleaseService) DeleteRelease(id int) error {
	err := s.repo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}

	return nil
}

// GetReleasesByArtist возвращает релизы по артисту (включая неактивных артистов)
func (s *ReleaseService) GetReleasesByArtist(artistID int) ([]model.Release, error) {
	releases, err := s.repo.GetByArtist(artistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get releases by artist ID %d: %w", artistID, err)
	}

	return releases, nil
}

// GetReleasesByGender возвращает релизы по полу (включая неактивных артистов)
func (s *ReleaseService) GetReleasesByGender(gender string) ([]model.Release, error) {
	var genderType model.Gender
	switch gender {
	case "female":
		genderType = model.GenderFemale
	case "male":
		genderType = model.GenderMale
	default:
		genderType = model.GenderMixed
	}

	releases, err := s.repo.GetByGender(genderType)
	if err != nil {
		return nil, fmt.Errorf("failed to get releases by gender %s: %w", gender, err)
	}

	return releases, nil
}

// GetAllReleases возвращает все релизы (включая неактивных артистов)
func (s *ReleaseService) GetAllReleases() ([]model.Release, error) {
	releases, err := s.repo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all releases: %w", err)
	}

	return releases, nil
}

// ParseReleasesForMonth парсит релизы за указанный месяц
func (s *ReleaseService) ParseReleasesForMonth(ctx context.Context, month string) (int, error) {
	s.logger.Info("Starting to parse releases", zap.String("month", month))

	// Получаем всех активных артистов для фильтрации
	artists, err := s.artistRepo.GetActive()
	if err != nil {
		return 0, fmt.Errorf("failed to get artists: %w", err)
	}

	// Создаем карту артистов для быстрого поиска
	artistMap := make(map[string]bool)
	for _, artist := range artists {
		artistMap[strings.ToLower(artist.Name)] = true
	}

	s.logger.Info("Found artists for filtering", zap.Int("count", len(artistMap)))

	// Логируем список активных артистов для отладки
	var artistNames []string
	for artistName := range artistMap {
		artistNames = append(artistNames, artistName)
	}
	s.logger.Info("Active artists list", zap.Strings("artists", artistNames))

	// Извлекаем год из строки месяца (формат: "september-2025" или "september")
	year := time.Now().Format("2006") // По умолчанию текущий год
	if strings.Contains(month, "-") {
		parts := strings.Split(month, "-")
		if len(parts) == 2 {
			month = parts[0]
			year = parts[1]
		}
	}

	// Сначала получаем ссылки на месячные страницы
	months := []string{month}
	links, err := s.scraper.FetchMonthlyLinks(ctx, months, year)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch monthly links: %w", err)
	}

	if len(links) == 0 {
		s.logger.Warn("No links found for month", zap.String("month", month))
		return 0, nil
	}

	// Парсим первую найденную ссылку
	url := links[0]
	s.logger.Info("Found monthly page URL", zap.String("month", month), zap.String("url", url))

	scrapedReleases, err := s.scraper.ParseMonthlyPage(ctx, url, month, year, artistMap)
	if err != nil {
		return 0, fmt.Errorf("failed to parse monthly page: %w", err)
	}

	s.logger.Info("Parsed releases from scraper", zap.Int("count", len(scrapedReleases)))

	// Конвертируем и сохраняем релизы
	savedCount := 0
	for _, scrapedRelease := range scrapedReleases {
		// Находим артиста в базе (GetByName теперь сам нормализует имя)
		artist, err := s.artistRepo.GetByName(scrapedRelease.Artist)
		if err != nil {
			s.logger.Warn("Failed to get artist from database",
				zap.String("artist", scrapedRelease.Artist),
				zap.Error(err))
			continue
		}

		// Если артист не найден, создаем нового
		if artist == nil {
			s.logger.Info("Artist not found in database, creating new",
				zap.String("artist", scrapedRelease.Artist))

			// Создаем нового артиста с полом "mixed" по умолчанию
			// Админ сможет потом изменить пол через команды
			newArtist := &model.Artist{
				Name:   scrapedRelease.Artist,
				Gender: model.GenderMixed, // По умолчанию mixed
			}

			err = s.artistRepo.Create(newArtist)
			if err != nil {
				s.logger.Warn("Failed to create new artist",
					zap.String("artist", scrapedRelease.Artist),
					zap.Error(err))
				continue
			}

			artist = newArtist
			s.logger.Info("Created new artist",
				zap.String("artist", scrapedRelease.Artist),
				zap.String("gender", artist.Gender.String()))
		} else {
			// Проверяем, нужно ли обновить имя артиста
			// Обновляем если имя отличается (включая регистр)
			if artist.Name != scrapedRelease.Artist {
				s.logger.Info("Updating artist name",
					zap.String("old_name", artist.Name),
					zap.String("new_name", scrapedRelease.Artist))

				artist.Name = scrapedRelease.Artist
				err = s.artistRepo.Update(artist)
				if err != nil {
					s.logger.Warn("Failed to update artist name",
						zap.String("artist", scrapedRelease.Artist),
						zap.Error(err))
				} else {
					s.logger.Info("Successfully updated artist name",
						zap.String("artist", scrapedRelease.Artist))
				}
			}
		}

		// Получаем тип релиза по умолчанию
		releaseType, err := s.releaseTypeRepo.GetByName("single")
		if err != nil {
			s.logger.Warn("Failed to get default release type", zap.Error(err))
			continue
		}
		if releaseType == nil {
			s.logger.Warn("Default release type not found")
			continue
		}

		// Создаем релиз
		release := &model.Release{
			ArtistID:      artist.ArtistID,
			ReleaseTypeID: releaseType.ReleaseTypeID,
			Title:         scrapedRelease.AlbumName, // Используем AlbumName как Title
			TitleTrack:    scrapedRelease.TitleTrack,
			AlbumName:     scrapedRelease.AlbumName,
			MV:            scrapedRelease.MV,
			Date:          scrapedRelease.Date,
			TimeMSK:       scrapedRelease.TimeMSK,
			IsActive:      true,
		}

		// Парсим дату для извлечения месяца и года
		if release.Date != "" {
			if parsedDate, err := s.utils.ParseReleaseDate(release.Date); err == nil {
				release.Month = s.utils.ExtractMonthFromDate(parsedDate)
				release.Year = s.utils.ExtractYearFromDate(parsedDate)
			}
		}

		// Сохраняем релиз
		err = s.CreateOrUpdateRelease(release)
		if err != nil {
			s.logger.Warn("Failed to save release",
				zap.String("artist", scrapedRelease.Artist),
				zap.String("title", scrapedRelease.AlbumName),
				zap.Error(err))
			continue
		}

		savedCount++
	}

	s.logger.Info("Completed parsing releases",
		zap.String("month", month),
		zap.Int("parsed", len(scrapedReleases)),
		zap.Int("saved", savedCount))

	return savedCount, nil
}

// GetReleasesByArtistName возвращает релизы по имени артиста
func (s *ReleaseService) GetReleasesByArtistName(artistName string) (string, error) {
	// Получаем релизы по имени артиста
	releases, err := s.repo.GetByArtistName(artistName)
	if err != nil {
		return "", fmt.Errorf("failed to get releases for artist %s: %w", artistName, err)
	}

	// Логируем результат поиска
	s.logger.Info("Search results for artist",
		zap.String("artist", artistName),
		zap.Int("count", len(releases)))

	// Форматируем ответ
	var result strings.Builder
	result.WriteString(fmt.Sprintf("🎵 Релизы артиста %s:\n\n", artistName))

	if len(releases) == 0 {
		result.WriteString("Релизы не найдены")
		return result.String(), nil
	}

	for _, release := range releases {
		var artistName string
		if release.Artist != nil {
			artistName = release.Artist.Name
		}

		// Форматируем релиз в нужном формате: дата | <b>АРТИСТ</b> | название (ссылка)
		line := fmt.Sprintf("%s | <b>%s</b>", release.Date, html.EscapeString(artistName))

		// Добавляем название релиза
		if release.Title != "" && release.Title != "N/A" {
			line += fmt.Sprintf(" | %s", html.EscapeString(release.Title))
		}

		// Добавляем ссылку на MV если есть (встраиваем в TitleTrack)
		if release.MV != "" && release.MV != "N/A" {
			// Очищаем TitleTrack
			cleanedTitleTrack := strings.ReplaceAll(release.TitleTrack, "Title Track:", "")
			cleanedTitleTrack = strings.TrimSpace(cleanedTitleTrack)

			if cleanedTitleTrack != "" && cleanedTitleTrack != "N/A" {
				// Встраиваем ссылку в название трека
				trackName := html.EscapeString(cleanedTitleTrack)
				line += fmt.Sprintf(" | <a href=\"%s\">%s</a>", release.MV, trackName)
			} else {
				// Если нет названия трека, добавляем просто ссылку
				line += fmt.Sprintf(" | <a href=\"%s\">Link</a>", release.MV)
			}
		} else if release.TitleTrack != "" && release.TitleTrack != "N/A" {
			// Если нет ссылки, но есть название трека, добавляем его
			cleanedTitleTrack := strings.ReplaceAll(release.TitleTrack, "Title Track:", "")
			cleanedTitleTrack = strings.TrimSpace(cleanedTitleTrack)
			line += fmt.Sprintf(" | %s", html.EscapeString(cleanedTitleTrack))
		}

		result.WriteString(line + "\n")
	}

	return result.String(), nil
}
