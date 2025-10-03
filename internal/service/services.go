// Package service содержит бизнес-логику приложения.
package service

import (
	"gemfactory/internal/config"
	"gemfactory/internal/external/scraper"
	"gemfactory/internal/external/spotify"
	"gemfactory/internal/model"
	"gemfactory/internal/storage"

	"github.com/uptrace/bun"
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
func NewServices(db *storage.Postgres, cfg *config.Config, logger *zap.Logger) *Services {
	configService := NewConfigService(db.GetDB(), logger)
	configLoader := NewConfigLoader(configService, logger)
	configLoader.LoadConfigFromDB(cfg)

	spotifyClient := NewSpotifyClient(cfg, logger)
	scraperClient := NewScraperClient(cfg, logger)
	playlistService := NewPlaylistServiceWithClient(db.GetDB(), spotifyClient, cfg.PlaylistURL, logger)

	coreServices := NewCoreServices(db, logger)
	coreServices.Release = NewReleaseService(db.GetDB(), scraperClient, logger)
	coreServices.Homework = NewHomeworkService(db.GetDB(), playlistService, coreServices.Task, logger)

	RegisterTaskExecutors(coreServices, configService, playlistService, logger)

	configWatcher := NewConfigWatcher(configService, coreServices.Task, coreServices.Scheduler, logger)

	return &Services{
		Artist:        coreServices.Artist,
		Release:       coreServices.Release,
		Homework:      coreServices.Homework,
		Playlist:      playlistService,
		Config:        configService,
		ConfigWatcher: configWatcher,
		Task:          coreServices.Task,
		Scheduler:     coreServices.Scheduler,
	}
}

// NewConfigLoader создает загрузчик конфигурации
func NewConfigLoader(configService *ConfigService, logger *zap.Logger) *config.ConfigLoader {
	return config.NewConfigLoader(configService, logger)
}

// NewSpotifyClient создает Spotify клиент
func NewSpotifyClient(cfg *config.Config, logger *zap.Logger) *spotify.Client {
	if cfg.SpotifyClientID == "" || cfg.SpotifyClientSecret == "" {
		logger.Warn("Spotify credentials not provided, Spotify client will not be created")
		return nil
	}

	client, err := spotify.NewClient(cfg.SpotifyClientID, cfg.SpotifyClientSecret, logger)
	if err != nil {
		logger.Error("Failed to create Spotify client", zap.Error(err))
		return nil
	}

	return client
}

// NewScraperClient создает скрейпер
func NewScraperClient(cfg *config.Config, logger *zap.Logger) scraper.Fetcher {
	scraperConfig := scraper.Config{
		HTTPClientConfig: scraper.HTTPClientConfig{
			MaxIdleConns:          cfg.ScraperConfig.HTTPClientConfig.MaxIdleConns,
			MaxIdleConnsPerHost:   cfg.ScraperConfig.HTTPClientConfig.MaxIdleConnsPerHost,
			IdleConnTimeout:       cfg.ScraperConfig.HTTPClientConfig.IdleConnTimeout,
			TLSHandshakeTimeout:   cfg.ScraperConfig.HTTPClientConfig.TLSHandshakeTimeout,
			ResponseHeaderTimeout: cfg.ScraperConfig.HTTPClientConfig.ResponseHeaderTimeout,
			DisableKeepAlives:     cfg.ScraperConfig.HTTPClientConfig.DisableKeepAlives,
		},
		RetryConfig: scraper.RetryConfig{
			MaxRetries:        cfg.ScraperConfig.RetryConfig.MaxRetries,
			InitialDelay:      cfg.ScraperConfig.RetryConfig.InitialDelay,
			MaxDelay:          cfg.ScraperConfig.RetryConfig.MaxDelay,
			BackoffMultiplier: cfg.ScraperConfig.RetryConfig.BackoffMultiplier,
		},
		RequestDelay: cfg.ScraperConfig.RequestDelay,
		LLMConfig: scraper.LLMConfig{
			BaseURL: cfg.LLMConfig.BaseURL,
			APIKey:  cfg.LLMConfig.APIKey,
			Timeout: cfg.LLMConfig.Timeout,
			Delay:   cfg.LLMConfig.Delay,
		},
	}
	return scraper.NewFetcher(scraperConfig, logger)
}

// NewPlaylistServiceWithClient создает сервис плейлиста с клиентом
func NewPlaylistServiceWithClient(db *bun.DB, spotifyClient *spotify.Client, playlistURL string, logger *zap.Logger) *PlaylistService {
	if spotifyClient == nil {
		logger.Warn("Spotify client not available, playlist service will not be created")
		return nil
	}
	return NewPlaylistService(db, *spotifyClient, playlistURL, logger)
}

// CoreServices содержит основные сервисы
type CoreServices struct {
	Artist    *ArtistService
	Release   *ReleaseService
	Homework  *HomeworkService
	Task      *TaskService
	Scheduler *Scheduler
}

// NewCoreServices создает основные сервисы
func NewCoreServices(db *storage.Postgres, logger *zap.Logger) *CoreServices {
	taskService := NewTaskService(db.GetDB(), logger)
	return &CoreServices{
		Artist:    NewArtistService(db.GetDB(), logger),
		Task:      taskService,
		Scheduler: NewScheduler(taskService, logger),
	}
}

// RegisterTaskExecutors регистрирует исполнителей задач
func RegisterTaskExecutors(coreServices *CoreServices, configService *ConfigService, playlistService *PlaylistService, logger *zap.Logger) {
	parseReleaseExecutor := NewParseReleaseTaskExecutor(coreServices.Release, logger)
	coreServices.Scheduler.RegisterExecutor(model.TaskTypeParseReleases, parseReleaseExecutor)

	homeworkResetExecutor := NewHomeworkResetTaskExecutor(coreServices.Homework, configService, logger)
	coreServices.Scheduler.RegisterExecutor(model.TaskTypeHomeworkReset, homeworkResetExecutor)

	if playlistService != nil {
		updatePlaylistExecutor := NewUpdatePlaylistTaskExecutor(playlistService, logger)
		coreServices.Scheduler.RegisterExecutor(model.TaskTypeUpdatePlaylist, updatePlaylistExecutor)
	} else {
		logger.Warn("Playlist service not available, update playlist tasks will not be registered")
	}
}
