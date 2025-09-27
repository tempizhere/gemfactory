// Package service содержит бизнес-логику приложения.
package service

import (
	"fmt"
	"gemfactory/internal/model"
	"gemfactory/internal/storage/repository"
	"sort"
	"strings"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ArtistService содержит бизнес-логику для работы с артистами
type ArtistService struct {
	repo   model.ArtistRepository
	logger *zap.Logger
}

// NewArtistService создает новый сервис артистов
func NewArtistService(db *bun.DB, logger *zap.Logger) *ArtistService {
	return &ArtistService{
		repo:   repository.NewArtistRepository(db, logger),
		logger: logger,
	}
}

// AddArtists добавляет артистов
func (s *ArtistService) AddArtists(artists []string, isFemale bool) (int, error) {
	addedCount := 0
	for _, artistName := range artists {
		// Проверяем, существует ли артист
		artist, err := s.repo.GetByName(artistName)
		if err != nil {
			return addedCount, fmt.Errorf("failed to get artist %s: %w", artistName, err)
		}

		// Если артист не существует, создаем его
		if artist == nil {
			artist = &model.Artist{
				Name:     artistName,
				Gender:   model.FromBool(isFemale),
				IsActive: true, // Новые артисты всегда активны
			}
			err = s.repo.Create(artist)
			if err != nil {
				return addedCount, fmt.Errorf("failed to create artist %s: %w", artistName, err)
			}
			addedCount++
		} else {
			// Если артист существует, обновляем его пол и активируем
			updated := false
			if artist.IsFemale() != isFemale {
				artist.SetGender(isFemale)
				updated = true
			}
			if !artist.IsActive {
				artist.IsActive = true
				updated = true
			}

			if updated {
				err = s.repo.Update(artist)
				if err != nil {
					return addedCount, fmt.Errorf("failed to update artist %s: %w", artistName, err)
				}
				addedCount++
			}
		}
	}

	return addedCount, nil
}

// RemoveArtists удаляет артистов (физическое удаление)
func (s *ArtistService) RemoveArtists(artists []string) (int, error) {
	removedCount := 0
	for _, artistName := range artists {
		// Получаем артиста
		artist, err := s.repo.GetByName(artistName)
		if err != nil {
			return removedCount, fmt.Errorf("failed to get artist %s: %w", artistName, err)
		}

		if artist == nil {
			s.logger.Warn("Artist not found", zap.String("artist", artistName))
			continue
		}

		// Удаляем артиста
		err = s.repo.Delete(artist.ArtistID)
		if err != nil {
			return removedCount, fmt.Errorf("failed to delete artist %s: %w", artistName, err)
		}
		removedCount++
	}

	return removedCount, nil
}

// DeactivateArtists деактивирует артистов (снимает флаг is_active)
func (s *ArtistService) DeactivateArtists(artists []string) (int, error) {
	deactivatedCount := 0
	for _, artistName := range artists {
		// Получаем артиста
		artist, err := s.repo.GetByName(artistName)
		if err != nil {
			return deactivatedCount, fmt.Errorf("failed to get artist %s: %w", artistName, err)
		}

		if artist == nil {
			s.logger.Warn("Artist not found", zap.String("artist", artistName))
			continue
		}

		// Если артист уже деактивирован, пропускаем
		if !artist.IsActive {
			s.logger.Info("Artist already deactivated", zap.String("artist", artistName))
			continue
		}

		// Деактивируем артиста
		artist.IsActive = false
		err = s.repo.Update(artist)
		if err != nil {
			return deactivatedCount, fmt.Errorf("failed to deactivate artist %s: %w", artistName, err)
		}
		deactivatedCount++
	}

	return deactivatedCount, nil
}

// GetFemaleArtists возвращает активных женских артистов
func (s *ArtistService) GetFemaleArtists() ([]string, error) {
	artists, err := s.repo.GetByGenderAndActive(model.GenderFemale, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get female artists: %w", err)
	}

	var names []string
	for _, artist := range artists {
		names = append(names, artist.Name)
	}

	return names, nil
}

// GetMaleArtists возвращает активных мужских артистов
func (s *ArtistService) GetMaleArtists() ([]string, error) {
	artists, err := s.repo.GetByGenderAndActive(model.GenderMale, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get male artists: %w", err)
	}

	var names []string
	for _, artist := range artists {
		names = append(names, artist.Name)
	}

	return names, nil
}

// GetAll возвращает всех артистов (включая неактивных)
func (s *ArtistService) GetAll() ([]model.Artist, error) {
	return s.repo.GetAll()
}

// GetAllActive возвращает только активных артистов
func (s *ArtistService) GetAllActive() ([]model.Artist, error) {
	return s.repo.GetActive()
}

// Export экспортирует данные всех артистов (включая неактивных)
func (s *ArtistService) Export() (string, error) {
	// Получаем всех артистов (включая неактивных)
	allArtists, err := s.GetAll()
	if err != nil {
		return "", fmt.Errorf("failed to get all artists: %w", err)
	}

	// Разделяем на женских и мужских
	var femaleArtists []string
	var maleArtists []string

	for _, artist := range allArtists {
		if artist.IsFemale() {
			femaleArtists = append(femaleArtists, artist.Name)
		} else {
			maleArtists = append(maleArtists, artist.Name)
		}
	}

	var response strings.Builder

	response.WriteString("<b>Женские артисты:</b>\n")
	if len(femaleArtists) == 0 {
		response.WriteString("пусто\n")
	} else {
		sort.Strings(femaleArtists)
		response.WriteString(fmt.Sprintf("<code>%s</code>\n", strings.Join(femaleArtists, ", ")))
	}

	// Добавляем перенос строки между категориями
	response.WriteString("\n")

	response.WriteString("<b>Мужские артисты:</b>\n")
	if len(maleArtists) == 0 {
		response.WriteString("пусто\n")
	} else {
		sort.Strings(maleArtists)
		response.WriteString(fmt.Sprintf("<code>%s</code>\n", strings.Join(maleArtists, ", ")))
	}

	// Добавляем подсчет в конце
	response.WriteString(fmt.Sprintf("\n📊 Всего артистов: %d\n💃 Женских: %d\n🤦‍♂️ Мужских: %d",
		len(femaleArtists)+len(maleArtists), len(femaleArtists), len(maleArtists)))

	return response.String(), nil
}

// FormatArtists форматирует артистов для отображения
func (s *ArtistService) FormatArtists() string {
	femaleArtists, _ := s.GetFemaleArtists()
	maleArtists, _ := s.GetMaleArtists()

	var response strings.Builder

	response.WriteString("<b>Женские артисты:</b>\n")
	if len(femaleArtists) == 0 {
		response.WriteString("пусто\n")
	} else {
		sort.Strings(femaleArtists)
		response.WriteString(fmt.Sprintf("<code>%s</code>\n", strings.Join(femaleArtists, ", ")))
	}

	// Добавляем перенос строки между категориями
	response.WriteString("\n")

	response.WriteString("<b>Мужские артисты:</b>\n")
	if len(maleArtists) == 0 {
		response.WriteString("пусто\n")
	} else {
		sort.Strings(maleArtists)
		response.WriteString(fmt.Sprintf("<code>%s</code>\n", strings.Join(maleArtists, ", ")))
	}

	// Добавляем подсчет в конце
	response.WriteString(fmt.Sprintf("\n📊 Всего артистов: %d\n💃 Женских: %d\n🤦‍♂️ Мужских: %d",
		len(femaleArtists)+len(maleArtists), len(femaleArtists), len(maleArtists)))

	return response.String()
}

// GetArtistCounts возвращает количество артистов по категориям
func (s *ArtistService) GetArtistCounts() (femaleCount, maleCount, totalCount int, err error) {
	// Получаем всех активных артистов
	artists, err := s.repo.GetActive()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get active artists: %w", err)
	}

	femaleCount = 0
	maleCount = 0

	for _, artist := range artists {
		switch artist.Gender {
		case model.GenderFemale:
			femaleCount++
		case model.GenderMale:
			maleCount++
		// GenderMixed не учитываем в подсчете
		}
	}

	totalCount = femaleCount + maleCount
	return femaleCount, maleCount, totalCount, nil
}
