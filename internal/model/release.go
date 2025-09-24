// Package model содержит модели данных.
//
// Группа: ENTITIES - Основные сущности
// Содержит: Release, ReleaseRepository, ScrapedReleaseData
package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Release представляет релиз
type Release struct {
	bun.BaseModel `bun:"table:releases"`

	ReleaseID     int       `bun:"release_id,pk,autoincrement" json:"release_id"`
	ArtistID      int       `bun:"artist_id,notnull" json:"artist_id"`
	ReleaseTypeID int       `bun:"release_type_id,notnull" json:"release_type_id"`
	Title         string    `bun:"title,notnull" json:"title"`
	TitleTrack    string    `bun:"title_track" json:"title_track"` // Название титульного трека
	AlbumName     string    `bun:"album_name" json:"album_name"`   // Название альбома
	MV            string    `bun:"mv" json:"mv"`                   // Ссылка на MV
	Date          string    `bun:"date,notnull" json:"date"`       // Дата релиза в формате DD.MM.YYYY
	TimeMSK       string    `bun:"time_msk" json:"time_msk"`       // Время в MSK
	Month         string    `bun:"month,notnull" json:"month"`
	Year          int       `bun:"year,notnull" json:"year"`
	IsActive      bool      `bun:"is_active,notnull,default:true" json:"is_active"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Связи
	Artist      *Artist           `bun:"rel:belongs-to,join:artist_id=artist_id" json:"artist,omitempty"`
	ReleaseType *ReleaseTypeModel `bun:"rel:belongs-to,join:release_type_id=release_type_id" json:"release_type,omitempty"`
}

// Validate проверяет валидность релиза
func (r *Release) Validate() error {
	var errors ValidationErrors

	if r.ArtistID <= 0 {
		errors = append(errors, ValidationError{Field: "artist_id", Message: "artist_id is required"})
	}

	if r.ReleaseTypeID <= 0 {
		errors = append(errors, ValidationError{Field: "release_type_id", Message: "release_type_id is required"})
	}

	if err := ValidateRequired("title", r.Title); err != nil {
		errors = append(errors, err.(ValidationError))
	}

	if err := ValidateRequired("date", r.Date); err != nil {
		errors = append(errors, err.(ValidationError))
	}

	if err := ValidateNonNegativeInt("year", r.Year); err != nil {
		errors = append(errors, err.(ValidationError))
	}

	if r.MV != "" {
		if err := ValidateURL("mv", r.MV); err != nil {
			errors = append(errors, err.(ValidationError))
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ReleaseRepository определяет интерфейс для работы с релизами
type ReleaseRepository interface {
	Repository[Release]
	GetByMonth(month string) ([]Release, error)
	GetByMonthAndYear(month string, year int) ([]Release, error)
	GetByGender(gender Gender) ([]Release, error)
	GetByType(releaseTypeID int) ([]Release, error)
	GetByMonthAndGender(month string, gender Gender) ([]Release, error)
	GetByMonthYearAndGender(month string, year int, gender Gender) ([]Release, error)
	GetByArtist(artistID int) ([]Release, error)
	GetByArtistName(artistName string) ([]Release, error)
	GetByDateRange(start, end time.Time) ([]Release, error)
	GetActive() ([]Release, error)
	GetByYear(year int) ([]Release, error)
	GetWithRelations() ([]Release, error)
	GetByArtistAndTitle(artistID int, title string) (*Release, error)
	GetByArtistDateAndTrack(artistID int, date, titleTrack string) (*Release, error)
	GetByArtistAndDate(artistID int, date string) (*Release, error)
}

// ScrapedReleaseData представляет данные релиза для скрейпера
type ScrapedReleaseData struct {
	Artist    string    `json:"artist"`
	Title     string    `json:"title"`
	Date      string    `json:"date"`
	Type      string    `json:"type"`
	Gender    string    `json:"gender"`
	Month     string    `json:"month"`
	Year      int       `json:"year"`
	ScrapedAt time.Time `json:"scraped_at"`
}

// ToScrapedReleaseData конвертирует Release в ScrapedReleaseData для совместимости со скрейпером
func (r *Release) ToScrapedReleaseData() ScrapedReleaseData {
	var artistName, typeName, genderName string

	if r.Artist != nil {
		artistName = r.Artist.Name
		genderName = r.Artist.Gender.String()
	}

	if r.ReleaseType != nil {
		typeName = r.ReleaseType.Name
	}

	return ScrapedReleaseData{
		Artist:    artistName,
		Title:     r.Title,
		Date:      r.Date,
		Type:      typeName,
		Gender:    genderName,
		Month:     r.Month,
		Year:      r.Year,
		ScrapedAt: r.CreatedAt,
	}
}

// FromScrapedReleaseData создает Release из ScrapedReleaseData
// Внимание: требует дополнительной обработки для получения artist_id и release_type_id
func FromScrapedReleaseData(sr ScrapedReleaseData) *Release {
	return &Release{
		Title:     sr.Title,
		Date:      sr.Date,
		Month:     sr.Month,
		Year:      sr.Year,
		IsActive:  true,
		CreatedAt: sr.ScrapedAt,
		UpdatedAt: sr.ScrapedAt,
	}
}

// GetDisplayTitle возвращает отображаемое название релиза
func (r *Release) GetDisplayTitle() string {
	if r.AlbumName != "" && r.AlbumName != "N/A" {
		return r.AlbumName
	}
	return r.Title
}

// GetDisplayTrack возвращает отображаемый трек
func (r *Release) GetDisplayTrack() string {
	if r.TitleTrack != "" && r.TitleTrack != "N/A" {
		return r.TitleTrack
	}
	return r.Title
}

// HasMV проверяет, есть ли ссылка на MV
func (r *Release) HasMV() bool {
	return r.MV != "" && r.MV != "N/A"
}

// GetFormattedDateTime возвращает отформатированную дату и время
func (r *Release) GetFormattedDateTime() string {
	if r.TimeMSK != "" && r.TimeMSK != "N/A" {
		return r.Date + " в " + r.TimeMSK
	}
	return r.Date
}

// IsValid проверяет валидность релиза
func (r *Release) IsValid() bool {
	return r.ArtistID > 0 && r.ReleaseTypeID > 0 && r.Title != "" && r.Date != ""
}
