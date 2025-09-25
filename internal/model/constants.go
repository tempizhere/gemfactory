// Package model содержит константы для моделей.
//
// Группа: BASE - Базовые компоненты
// Содержит: ReleaseType, Gender, HomeworkStatus, ConfigKey
package model

// ReleaseType представляет тип релиза
type ReleaseType string

const (
	ReleaseTypeSingle ReleaseType = "single"
	ReleaseTypeAlbum  ReleaseType = "album"
	ReleaseTypeEP     ReleaseType = "ep"
)

// Gender представляет пол артиста
type Gender string

const (
	GenderFemale Gender = "female"
	GenderMale   Gender = "male"
	GenderMixed  Gender = "mixed"
)

// String возвращает строковое представление пола
func (g Gender) String() string {
	return string(g)
}

// IsValid проверяет валидность пола
func (g Gender) IsValid() bool {
	switch g {
	case GenderFemale, GenderMale, GenderMixed:
		return true
	default:
		return false
	}
}

// ToBool конвертирует пол в boolean (true для женского)
func (g Gender) ToBool() bool {
	return g == GenderFemale
}

// FromBool создает Gender из boolean
func FromBool(isFemale bool) Gender {
	if isFemale {
		return GenderFemale
	}
	return GenderMale
}

// HomeworkStatus представляет статус домашнего задания
type HomeworkStatus string

const (
	HomeworkStatusActive    HomeworkStatus = "active"
	HomeworkStatusCompleted HomeworkStatus = "completed"
	HomeworkStatusExpired   HomeworkStatus = "expired"
)

// String возвращает строковое представление статуса
func (hs HomeworkStatus) String() string {
	return string(hs)
}

// IsValid проверяет валидность статуса
func (hs HomeworkStatus) IsValid() bool {
	switch hs {
	case HomeworkStatusActive, HomeworkStatusCompleted, HomeworkStatusExpired:
		return true
	default:
		return false
	}
}

// ConfigKey представляет ключи конфигурации
type ConfigKey string

const (
	ConfigKeySpotifyClientID     ConfigKey = "spotify_client_id"
	ConfigKeySpotifyClientSecret ConfigKey = "spotify_client_secret"
	ConfigKeyPlaylistURL         ConfigKey = "playlist_url"
	ConfigKeyBotToken            ConfigKey = "bot_token"
	ConfigKeyAdminUsername       ConfigKey = "admin_username"
	ConfigKeyTimezone            ConfigKey = "timezone"
	ConfigKeyHealthPort          ConfigKey = "health_port"
	ConfigKeyLogLevel            ConfigKey = "log_level"
)
