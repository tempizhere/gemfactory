// Package app содержит фабрику компонентов приложения.
package app

import (
	"fmt"
	"gemfactory/internal/config"
	"gemfactory/internal/external/scraper"
	"gemfactory/internal/external/spotify"
	"gemfactory/internal/external/telegram"
	"gemfactory/internal/health"
	"gemfactory/internal/middleware"
	"gemfactory/internal/service"
	"gemfactory/internal/storage"
	"os"

	"go.uber.org/zap"
)

// ComponentFactory создает компоненты приложения
type ComponentFactory struct {
	config *config.Config
	logger *zap.Logger
}

// NewComponentFactory создает новую фабрику компонентов
func NewComponentFactory(config *config.Config, logger *zap.Logger) *ComponentFactory {
	if config == nil {
		logger.Fatal("Config cannot be nil")
	}
	if logger == nil {
		panic("Logger cannot be nil")
	}

	return &ComponentFactory{
		config: config,
		logger: logger,
	}
}

// CreateDatabase создает подключение к базе данных
func (f *ComponentFactory) CreateDatabase() (*storage.Postgres, error) {
	if f.config.DatabaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	db, err := storage.NewPostgres(f.config.DatabaseURL, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	f.logger.Info("Database connection created successfully")
	return db, nil
}

// CreateTelegramClient создает клиент Telegram
func (f *ComponentFactory) CreateTelegramClient() (*telegram.Client, error) {
	if f.config.BotToken == "" {
		return nil, fmt.Errorf("bot token is required")
	}

	client, err := telegram.NewClient(f.config.BotToken, f.config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram client: %w", err)
	}

	f.logger.Info("Telegram client created successfully")
	return client, nil
}

// CreateScraper создает скрейпер
func (f *ComponentFactory) CreateScraper() scraper.Fetcher {
	scraperConfig := scraper.Config{
		HTTPClientConfig: scraper.HTTPClientConfig{
			MaxIdleConns:          f.config.ScraperConfig.HTTPClientConfig.MaxIdleConns,
			MaxIdleConnsPerHost:   f.config.ScraperConfig.HTTPClientConfig.MaxIdleConnsPerHost,
			IdleConnTimeout:       f.config.ScraperConfig.HTTPClientConfig.IdleConnTimeout,
			TLSHandshakeTimeout:   f.config.ScraperConfig.HTTPClientConfig.TLSHandshakeTimeout,
			ResponseHeaderTimeout: f.config.ScraperConfig.HTTPClientConfig.ResponseHeaderTimeout,
			DisableKeepAlives:     f.config.ScraperConfig.HTTPClientConfig.DisableKeepAlives,
		},
		RetryConfig: scraper.RetryConfig{
			MaxRetries:        f.config.ScraperConfig.RetryConfig.MaxRetries,
			InitialDelay:      f.config.ScraperConfig.RetryConfig.InitialDelay,
			MaxDelay:          f.config.ScraperConfig.RetryConfig.MaxDelay,
			BackoffMultiplier: f.config.ScraperConfig.RetryConfig.BackoffMultiplier,
		},
		RequestDelay: f.config.ScraperConfig.RequestDelay,
		LLMConfig: scraper.LLMConfig{
			BaseURL: f.config.LLMConfig.BaseURL,
			APIKey:  f.config.LLMConfig.APIKey,
			Timeout: f.config.LLMConfig.Timeout,
		},
	}
	scraperInstance := scraper.NewFetcher(scraperConfig, f.logger)
	f.logger.Info("Scraper created successfully")
	return scraperInstance
}

// CreateSpotifyClient создает Spotify клиент
func (f *ComponentFactory) CreateSpotifyClient() (*spotify.Client, error) {
	// Проверяем, есть ли настройки Spotify
	if f.config.SpotifyClientID == "" || f.config.SpotifyClientSecret == "" {
		f.logger.Warn("Spotify credentials not provided, Spotify client will not be created")
		return nil, fmt.Errorf("spotify client ID and secret are required")
	}

	client, err := spotify.NewClient(f.config.SpotifyClientID, f.config.SpotifyClientSecret, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create spotify client: %w", err)
	}

	f.logger.Info("Spotify client created successfully")
	return client, nil
}

// CreateServices создает все сервисы
func (f *ComponentFactory) CreateServices(db *storage.Postgres) (*service.Services, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	services := service.NewServices(db, f.config, f.logger)
	f.logger.Info("Services created successfully")
	return services, nil
}

// CreateMiddleware создает middleware
func (f *ComponentFactory) CreateMiddleware() *middleware.Middleware {
	middlewareManager := middleware.New(f.config, f.logger)
	f.logger.Info("Middleware created successfully")
	return middlewareManager
}

// CreateHealthServer создает сервер health check
func (f *ComponentFactory) CreateHealthServer(db *storage.Postgres) (*health.Server, error) {
	if !f.config.HealthCheckEnabled {
		f.logger.Info("Health check server is disabled")
		return nil, nil
	}

	if f.config.HealthPort == "" {
		return nil, fmt.Errorf("health port is required when health check is enabled")
	}

	server := health.NewServer(f.config.HealthPort, f.logger, db)
	f.logger.Info("Health check server created", zap.String("port", f.config.HealthPort))
	return server, nil
}

// CreateAppDataDirectory создает директорию данных приложения
func (f *ComponentFactory) CreateAppDataDirectory() error {
	dataDir := f.config.GetAppDataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		f.logger.Error("Failed to create app data directory", zap.String("dir", dataDir), zap.Error(err))
		return fmt.Errorf("failed to create app data directory: %w", err)
	}
	f.logger.Info("App data directory ready", zap.String("dir", dataDir))
	return nil
}

// CreateBot создает полный экземпляр бота со всеми зависимостями
func (f *ComponentFactory) CreateBot() (*Bot, error) {
	// Создаем директорию данных приложения
	if err := f.CreateAppDataDirectory(); err != nil {
		return nil, fmt.Errorf("failed to create app data directory: %w", err)
	}

	// Создаем базу данных
	db, err := f.CreateDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Создаем сервисы
	services, err := f.CreateServices(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create services: %w", err)
	}

	// Создаем Telegram клиент
	tgClient, err := f.CreateTelegramClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram client: %w", err)
	}

	// Создаем health check сервер
	healthServer, err := f.CreateHealthServer(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create health server: %w", err)
	}

	// Создаем middleware
	middlewareManager := f.CreateMiddleware()

	// Создаем бота
	bot, err := NewBot(f.config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	// Устанавливаем компоненты в бота
	bot.db = db
	bot.telegram = tgClient
	bot.health = healthServer
	bot.services = services
	bot.middleware = middlewareManager

	// Проверяем наличие артистов
	femaleArtists, _ := services.Artist.GetFemaleArtists()
	maleArtists, _ := services.Artist.GetMaleArtists()
	if len(femaleArtists) == 0 && len(maleArtists) == 0 {
		f.logger.Warn("No artists found; add artists using /add_artist")
	}

	f.logger.Info("Bot created successfully with all dependencies")
	return bot, nil
}

// ValidateConfig проверяет конфигурацию на корректность
func (f *ComponentFactory) ValidateConfig() error {
	if f.config == nil {
		return fmt.Errorf("config is nil")
	}

	// Проверяем обязательные поля
	if f.config.BotToken == "" {
		return fmt.Errorf("bot token is required")
	}
	if f.config.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	if f.config.HealthCheckEnabled && f.config.HealthPort == "" {
		return fmt.Errorf("health port is required when health check is enabled")
	}

	// Проверяем опциональные поля
	if f.config.SpotifyClientID != "" && f.config.SpotifyClientSecret == "" {
		return fmt.Errorf("spotify client secret is required when client ID is provided")
	}
	if f.config.SpotifyClientSecret != "" && f.config.SpotifyClientID == "" {
		return fmt.Errorf("spotify client ID is required when client secret is provided")
	}

	f.logger.Info("Configuration validation passed")
	return nil
}

// GetAppDataDir возвращает директорию данных приложения
func (f *ComponentFactory) GetAppDataDir() string {
	return f.config.GetAppDataDir()
}
