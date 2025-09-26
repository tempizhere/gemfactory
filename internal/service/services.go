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
func NewServices(db *storage.Postgres, cfg *config.Config, logger *zap.Logger) *Services {
	configService := NewConfigService(db.GetDB(), logger)

	// Создаем загрузчик конфигурации
	configLoader := config.NewConfigLoader(configService, logger)

	// Загружаем недостающие значения из базы данных (приоритет: env > база данных)
	// ADMIN_USERNAME
	configLoader.LoadConfigValueWithSetter(cfg.AdminUsername, "ADMIN_USERNAME", func(value string) {
		cfg.AdminUsername = value
	})

	// LLM_API_KEY
	llmAPIKey := configLoader.LoadConfigValueWithSetter(cfg.LLMConfig.APIKey, "LLM_API_KEY", func(value string) {
		cfg.LLMConfig.APIKey = value
	})

	// BOT_TOKEN
	configLoader.LoadConfigValueWithSetter(cfg.BotToken, "BOT_TOKEN", func(value string) {
		cfg.BotToken = value
	})

	// SPOTIFY_CLIENT_ID
	spotifyClientID := configLoader.LoadConfigValueWithSetter(cfg.SpotifyClientID, "SPOTIFY_CLIENT_ID", func(value string) {
		cfg.SpotifyClientID = value
	})

	// SPOTIFY_CLIENT_SECRET
	spotifyClientSecret := configLoader.LoadConfigValueWithSetter(cfg.SpotifyClientSecret, "SPOTIFY_CLIENT_SECRET", func(value string) {
		cfg.SpotifyClientSecret = value
	})

	// PLAYLIST_URL
	playlistURL := configLoader.LoadConfigValueWithSetter(cfg.PlaylistURL, "PLAYLIST_URL", func(value string) {
		cfg.PlaylistURL = value
	})

	// Создаем Spotify клиент с обновленными значениями
	spotifyClient, err := spotify.NewClient(spotifyClientID, spotifyClientSecret, logger)
	if err != nil {
		logger.Error("Failed to create Spotify client", zap.Error(err))
		// Продолжаем без Spotify клиента
		spotifyClient = nil
	}

	// Создаем скрейпер с LLM_API_KEY из базы данных
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
			APIKey:  llmAPIKey, // Используем ключ из базы данных
			Timeout: cfg.LLMConfig.Timeout,
			Delay:   cfg.LLMConfig.Delay,
		},
	}
	scraperClient := scraper.NewFetcher(scraperConfig, logger)

	// Создаем playlistService только если Spotify клиент доступен
	var playlistService *PlaylistService
	if spotifyClient != nil {
		playlistService = NewPlaylistService(db.GetDB(), *spotifyClient, playlistURL, logger)
	} else {
		logger.Warn("Spotify client not available, playlist service will not be created")
	}

	// Создаем остальные сервисы
	artistService := NewArtistService(db.GetDB(), logger)
	releaseService := NewReleaseService(db.GetDB(), scraperClient, logger)
	taskService := NewTaskService(db.GetDB(), logger)
	homeworkService := NewHomeworkService(db.GetDB(), playlistService, taskService, logger)

	// Создаем планировщик
	scheduler := NewScheduler(taskService, logger)

	// Регистрируем исполнителей задач
	parseReleaseExecutor := NewParseReleaseTaskExecutor(releaseService, logger)
	scheduler.RegisterExecutor(model.TaskTypeParseReleases, parseReleaseExecutor)

	// Регистрируем updatePlaylistExecutor только если playlistService доступен
	if playlistService != nil {
		updatePlaylistExecutor := NewUpdatePlaylistTaskExecutor(playlistService, logger)
		scheduler.RegisterExecutor(model.TaskTypeUpdatePlaylist, updatePlaylistExecutor)
	} else {
		logger.Warn("Playlist service not available, update playlist tasks will not be registered")
	}

	homeworkResetExecutor := NewHomeworkResetTaskExecutor(homeworkService, configService, logger)
	scheduler.RegisterExecutor(model.TaskTypeHomeworkReset, homeworkResetExecutor)

	configWatcher := NewConfigWatcher(configService, taskService, scheduler, logger)

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
