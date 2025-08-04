// Package playlist содержит типы данных для работы с плейлистами.
package playlist

// Track представляет трек из плейлиста
type Track struct {
	ID     string // Spotify Track ID
	Title  string // Название трека
	Artist string // Исполнитель
}

// PlaylistService определяет интерфейс для работы с плейлистами
type PlaylistService interface {
	// GetRandomTrack возвращает случайный трек из плейлиста
	GetRandomTrack() (*Track, error)

	// GetTotalTracks возвращает общее количество треков в плейлисте
	GetTotalTracks() int

	// LoadPlaylist загружает плейлист из CSV файла
	LoadPlaylist(filePath string) error

	// IsLoaded проверяет, загружен ли плейлист
	IsLoaded() bool
}

// PlaylistManager определяет интерфейс для управления плейлистами
type PlaylistManager interface {
	// GetRandomTrack возвращает случайный трек из плейлиста
	GetRandomTrack() (*Track, error)

	// GetTotalTracks возвращает общее количество треков в плейлисте
	GetTotalTracks() int

	// LoadPlaylistFromFile загружает плейлист из файла
	LoadPlaylistFromFile(filePath string) error

	// SavePlaylistToFile сохраняет плейлист в файл
	SavePlaylistToFile(filePath string) error

	// LoadPlaylistFromStorage загружает плейлист из постоянного хранилища
	LoadPlaylistFromStorage() error

	// SavePlaylistToStorage сохраняет плейлист в постоянное хранилище
	SavePlaylistToStorage() error

	// IsLoaded проверяет, загружен ли плейлист
	IsLoaded() bool

	// Clear очищает плейлист
	Clear()
}
