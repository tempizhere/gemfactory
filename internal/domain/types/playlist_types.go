// Package types содержит типы для работы с плейлистами.
package types

import (
	"time"

	"gemfactory/internal/gateway/spotify"

	"go.uber.org/zap"
)

// PlaylistServiceInterface определяет интерфейс для работы с плейлистами
type PlaylistServiceInterface interface {
	GetRandomTrack() (*spotify.Track, error)
	GetTotalTracks() int
	LoadPlaylistFromSpotify(playlistURL string) error
	IsLoaded() bool
}

// PlaylistManagerInterface определяет интерфейс для управления плейлистами
type PlaylistManagerInterface interface {
	GetRandomTrack() (*spotify.Track, error)
	GetTotalTracks() int
	GetPlaylistInfo() (*spotify.PlaylistInfo, error)
	LoadPlaylistFromSpotify(playlistURL string) error
	LoadPlaylistFromStorage() error
	SavePlaylistToStorage() error
	IsLoaded() bool
	Clear()
}

// PlaylistSchedulerInterface определяет интерфейс для планировщика плейлистов
type PlaylistSchedulerInterface interface {
	Start()
	Stop()
	GetLastUpdateTime() time.Time
	GetNextUpdateTime() time.Time
}

// HomeworkInfo содержит информацию о выданном домашнем задании
type HomeworkInfo struct {
	RequestTime time.Time
	Track       *spotify.Track
	PlayCount   int
}

// HomeworkCacheInterface определяет интерфейс для кэша домашних заданий
type HomeworkCacheInterface interface {
	CanRequest(userID int64) bool
	RecordRequest(userID int64, track *spotify.Track, playCount int)
	GetTimeUntilNextRequest(userID int64) time.Duration
	Cleanup()
	GetTotalRequests() int
	GetUniqueUsers() int
	GetHomeworkInfo(userID int64) *HomeworkInfo
	SetStoragePath(path string)
	SetLogger(logger *zap.Logger)
	LoadFromStorage() error
	SaveToStorage() error
}
