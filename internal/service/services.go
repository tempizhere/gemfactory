// Package service содержит бизнес-логику приложения.
package service

import (
	"gemfactory/internal/config"
	"gemfactory/internal/external/scraper"
	"gemfactory/internal/external/spotify"
	"gemfactory/internal/model"
	"gemfactory/internal/storage"

	"go.uber.org/zap"
)

// Services содержит все сервисы приложения
type Services struct {
	Artist        *ArtistService
	Release       *ReleaseService
	Homework      *HomeworkService
	Playlist      *PlaylistService
	Config        *ConfigService
	ConfigWatcher *ConfigWatcher
	Task          *TaskService
	Scheduler     *Scheduler
}

// NewServices создает все сервисы
func NewServices(db *storage.Postgres, config *config.Config, logger *zap.Logger) *Services {
	// Создаем Spotify клиент
	spotifyClient, err := spotify.NewClient(config.SpotifyClientID, config.SpotifyClientSecret, logger)
	if err != nil {
		logger.Error("Failed to create Spotify client", zap.Error(err))
		// Продолжаем без Spotify клиента
		spotifyClient = nil
	}

	// Создаем configService сначала для загрузки конфигурации из БД
	configService := NewConfigService(db.GetDB(), logger)

	// Загружаем LLM_API_KEY из базы данных
	llmAPIKey := config.LLMConfig.APIKey // Fallback на переменные окружения
	if dbAPIKey, err := configService.Get("LLM_API_KEY"); err == nil {
		llmAPIKey = dbAPIKey
		logger.Info("Loaded LLM_API_KEY from database")
	} else {
		logger.Debug("Failed to load LLM_API_KEY from database, using env var", zap.Error(err))
	}

	// Создаем скрейпер с LLM_API_KEY из базы данных
	scraperConfig := scraper.Config{
		HTTPClientConfig: scraper.HTTPClientConfig{
			MaxIdleConns:          config.ScraperConfig.HTTPClientConfig.MaxIdleConns,
			MaxIdleConnsPerHost:   config.ScraperConfig.HTTPClientConfig.MaxIdleConnsPerHost,
			IdleConnTimeout:       config.ScraperConfig.HTTPClientConfig.IdleConnTimeout,
			TLSHandshakeTimeout:   config.ScraperConfig.HTTPClientConfig.TLSHandshakeTimeout,
			ResponseHeaderTimeout: config.ScraperConfig.HTTPClientConfig.ResponseHeaderTimeout,
			DisableKeepAlives:     config.ScraperConfig.HTTPClientConfig.DisableKeepAlives,
		},
		RetryConfig: scraper.RetryConfig{
			MaxRetries:        config.ScraperConfig.RetryConfig.MaxRetries,
			InitialDelay:      config.ScraperConfig.RetryConfig.InitialDelay,
			MaxDelay:          config.ScraperConfig.RetryConfig.MaxDelay,
			BackoffMultiplier: config.ScraperConfig.RetryConfig.BackoffMultiplier,
		},
		RequestDelay: config.ScraperConfig.RequestDelay,
		LLMConfig: scraper.LLMConfig{
			BaseURL: config.LLMConfig.BaseURL,
			APIKey:  llmAPIKey, // Используем ключ из базы данных
			Timeout: config.LLMConfig.Timeout,
		},
	}
	scraperClient := scraper.NewFetcher(scraperConfig, logger)

	// Создаем остальные сервисы
	artistService := NewArtistService(db.GetDB(), logger)
	releaseService := NewReleaseService(db.GetDB(), scraperClient, logger)
	homeworkService := NewHomeworkService(db.GetDB(), logger)
	playlistService := NewPlaylistService(db.GetDB(), *spotifyClient, logger)
	taskService := NewTaskService(db.GetDB(), logger)

	// Создаем планировщик
	scheduler := NewScheduler(taskService, logger)

	// Регистрируем исполнителей задач
	parseReleaseExecutor := NewParseReleaseTaskExecutor(releaseService, logger)
	scheduler.RegisterExecutor(model.TaskTypeParseReleases, parseReleaseExecutor)

	updatePlaylistExecutor := NewUpdatePlaylistTaskExecutor(playlistService, configService, logger)
	scheduler.RegisterExecutor(model.TaskTypeUpdatePlaylist, updatePlaylistExecutor)

	homeworkResetExecutor := NewHomeworkResetTaskExecutor(homeworkService, configService, logger)
	scheduler.RegisterExecutor(model.TaskTypeHomeworkReset, homeworkResetExecutor)

	// Создаем наблюдатель конфигурации
	configWatcher := NewConfigWatcher(configService, taskService, logger)

	return &Services{
		Artist:        artistService,
		Release:       releaseService,
		Homework:      homeworkService,
		Playlist:      playlistService,
		Config:        configService,
		ConfigWatcher: configWatcher,
		Task:          taskService,
		Scheduler:     scheduler,
	}
}
