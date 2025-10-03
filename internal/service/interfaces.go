package service

import (
	"gemfactory/internal/external/spotify"
	"gemfactory/internal/model"
)

// PlaylistServiceInterface определяет интерфейс для работы с плейлистами
type PlaylistServiceInterface interface {
	ReloadPlaylist() error
	GetPlaylistTracks() ([]model.PlaylistTracks, error)
	UpdatePlaylist() error
	GetPlaylistInfo() (*spotify.PlaylistInfo, error)
}

// TaskServiceInterface определяет интерфейс для работы с задачами
type TaskServiceInterface interface {
	CreateTask(task *model.Task) error
	UpdateTask(task *model.Task) error
	DeleteTask(taskID int) error
	GetAllTasks() ([]model.Task, error)
	GetActiveTasks() ([]model.Task, error)
	GetDueTasks() ([]model.Task, error)
	UpdateRunStats(taskID int, success bool, err error) error
	GetTasksByType(taskType model.TaskType) ([]model.Task, error)
	GetByName(name string) (*model.Task, error)
}

// ConfigServiceInterface определяет интерфейс для работы с конфигурацией
type ConfigServiceInterface interface {
	GetConfigValue(key string) (string, error)
	SetConfigValue(key, value string) error
	GetAllConfig() (map[string]string, error)
	Get(key string) (string, error)
	GetAll() (string, error)
}

// SchedulerInterface определяет интерфейс для планировщика задач
type SchedulerInterface interface {
	Start() error
	Stop()
	RegisterExecutor(taskType model.TaskType, executor TaskExecutor)
	ReloadTasks() error
}
