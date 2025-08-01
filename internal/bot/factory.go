package bot

import (
	"fmt"
	"gemfactory/internal/bot/keyboard"
	"gemfactory/internal/bot/middleware"
	"gemfactory/internal/bot/service"
	"gemfactory/internal/config"
	"gemfactory/internal/domain/artist"
	"gemfactory/internal/domain/playlist"
	"gemfactory/internal/domain/types"
	"gemfactory/internal/gateway/scraper"
	"gemfactory/internal/gateway/telegram/botapi"
	releasecache "gemfactory/internal/infrastructure/cache"
	"gemfactory/internal/infrastructure/debounce"
	"gemfactory/internal/infrastructure/health"
	"gemfactory/internal/infrastructure/metrics"
	"gemfactory/internal/infrastructure/updater"
	"gemfactory/internal/infrastructure/worker"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// ComponentFactory создает компоненты бота
type ComponentFactory struct {
	config *config.Config // Используем конкретный тип вместо интерфейса
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

// CreateBotAPI создает API для работы с Telegram
func (f *ComponentFactory) CreateBotAPI() (botapi.BotAPI, error) {
	if f.config.BotToken == "" {
		return nil, fmt.Errorf("bot token is required")
	}

	tgAPI, err := tgbotapi.NewBotAPI(f.config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot API: %w", err)
	}

	api := botapi.NewTelegramBotAPI(tgAPI, f.logger)

	// Логируем детальную информацию о Telegram Bot API
	f.logger.Info("Telegram Bot API created successfully",
		zap.String("bot_username", tgAPI.Self.UserName),
		zap.String("bot_first_name", tgAPI.Self.FirstName),
		zap.Int64("bot_id", tgAPI.Self.ID),
		zap.Bool("debug_mode", tgAPI.Debug))

	return api, nil
}

// CreateWhitelistManager создает менеджер белых списков
func (f *ComponentFactory) CreateWhitelistManager() artist.WhitelistManager {
	manager := artist.NewWhitelistManager(f.config.GetAppDataDir(), f.logger)

	// Логируем информацию о загруженных списках
	femaleCount := len(manager.GetFemaleWhitelist())
	maleCount := len(manager.GetMaleWhitelist())

	f.logger.Info("Whitelist manager created",
		zap.Int("female_artists", femaleCount),
		zap.Int("male_artists", maleCount),
		zap.Int("total_artists", femaleCount+maleCount))

	return manager
}

// CreateScraper создает скрейпер
func (f *ComponentFactory) CreateScraper() scraper.Fetcher {
	scraper := scraper.NewFetcher(f.config, f.logger)
	f.logger.Info("Scraper created successfully")
	return scraper
}

// CreateManager создает менеджер кэша
func (f *ComponentFactory) CreateManager(
	whitelistManager artist.WhitelistManager,
	scraper scraper.Fetcher,
	metrics metrics.Interface,
	workerPool worker.PoolInterface,
) (releasecache.Cache, error) {
	if whitelistManager == nil {
		return nil, fmt.Errorf("whitelist manager is required")
	}
	if scraper == nil {
		return nil, fmt.Errorf("scraper is required")
	}
	if workerPool == nil {
		return nil, fmt.Errorf("worker pool is required")
	}

	manager := releasecache.NewManager(f.config, f.logger, whitelistManager, scraper, nil, workerPool)

	// Создаем updater и устанавливаем его в manager
	updater := updater.NewUpdater(f.config, f.logger, whitelistManager, manager, scraper)
	manager.SetUpdater(updater)

	// Устанавливаем метрики если они переданы
	if metrics != nil {
		manager.SetMetrics(metrics)
		updater.SetMetrics(metrics)
		f.logger.Debug("Metrics set in cache manager and updater")
	}

	f.logger.Info("Cache manager created successfully")
	return manager, nil
}

// CreateServices создает сервисы
func (f *ComponentFactory) CreateServices(
	whitelistManager artist.WhitelistManager,
	cache releasecache.Cache,
) (*service.ReleaseService, *service.ArtistService, error) {
	if whitelistManager == nil {
		return nil, nil, fmt.Errorf("whitelist manager is required")
	}
	if cache == nil {
		return nil, nil, fmt.Errorf("cache is required")
	}

	releaseService := service.NewReleaseService(whitelistManager, f.config, f.logger, cache)
	artistService := service.NewArtistService(whitelistManager, f.logger)

	f.logger.Info("Services created successfully")
	return releaseService, artistService, nil
}

// CreateKeyboardManager создает менеджер клавиатуры
func (f *ComponentFactory) CreateKeyboardManager(
	api botapi.BotAPI,
	whitelistManager artist.WhitelistManager,
	cache releasecache.Cache,
	workerPool worker.PoolInterface,
) (keyboard.ManagerInterface, error) {
	if api == nil {
		return nil, fmt.Errorf("bot API is required")
	}
	if whitelistManager == nil {
		return nil, fmt.Errorf("whitelist manager is required")
	}
	if cache == nil {
		return nil, fmt.Errorf("cache is required")
	}
	if workerPool == nil {
		return nil, fmt.Errorf("worker pool is required")
	}

	manager := keyboard.NewKeyboardManager(api, f.logger, whitelistManager, f.config, cache, workerPool)
	f.logger.Info("Keyboard manager created successfully")
	return manager, nil
}

// CreateWorkerPool создает пул воркеров
func (f *ComponentFactory) CreateWorkerPool() worker.PoolInterface {
	maxWorkers := f.config.MaxConcurrentRequests
	if maxWorkers <= 0 {
		maxWorkers = 5 // значение по умолчанию
	}

	pool := worker.NewWorkerPool(maxWorkers, 100, f.logger)
	f.logger.Info("Worker pool created", zap.Int("max_workers", maxWorkers))
	return pool
}

// CreateRateLimiter создает ограничитель запросов
func (f *ComponentFactory) CreateRateLimiter() middleware.RateLimiterInterface {
	if !f.config.RateLimitEnabled {
		f.logger.Info("Rate limiting is disabled")
		return nil
	}

	limiter := middleware.NewRateLimiter(
		f.config.RateLimitRequests,
		f.config.RateLimitWindow,
		f.logger,
	)

	f.logger.Info("Rate limiter created",
		zap.Int("requests", f.config.RateLimitRequests),
		zap.Duration("window", f.config.RateLimitWindow))

	return limiter
}

// CreateServer создает сервер health check
func (f *ComponentFactory) CreateServer(botAPI botapi.BotAPI, cache releasecache.Cache, workerPool worker.PoolInterface) health.ServerInterface {
	if !f.config.HealthCheckEnabled {
		f.logger.Info("Health check server is disabled")
		return nil
	}

	server := health.NewHealthServer(f.config.HealthCheckPort, f.logger, botAPI, cache, workerPool)
	f.logger.Info("Health check server created", zap.Int("port", f.config.HealthCheckPort))
	return server
}

// CreateMetrics создает систему метрик
func (f *ComponentFactory) CreateMetrics() metrics.Interface {
	if !f.config.MetricsEnabled {
		f.logger.Info("Metrics are disabled")
		return nil
	}

	m := metrics.NewMetrics(f.logger)
	f.logger.Info("Metrics system created")
	return m
}

// CreateDependencies создает все зависимости
func (f *ComponentFactory) CreateDependencies() (*types.Dependencies, error) {
	// Создаем директорию данных приложения
	dataDir := f.config.GetAppDataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		f.logger.Error("Failed to create app data directory", zap.String("dir", dataDir), zap.Error(err))
		return nil, fmt.Errorf("failed to create app data directory: %w", err)
	}
	f.logger.Info("App data directory ready", zap.String("dir", dataDir))

	// Создаем debouncer
	debouncer := debounce.NewDebouncer()

	// Создаем метрики
	metrics := f.CreateMetrics()

	// Создаем остальные компоненты
	api, err := f.CreateBotAPI()
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	whitelistManager := f.CreateWhitelistManager()
	scraper := f.CreateScraper()
	workerPool := f.CreateWorkerPool()

	// Создаем cache manager
	cache, err := f.CreateManager(whitelistManager, scraper, metrics, workerPool)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	// Создаем сервисы
	releaseService, artistService, err := f.CreateServices(whitelistManager, cache)
	if err != nil {
		return nil, fmt.Errorf("failed to create services: %w", err)
	}

	// Создаем keyboard manager
	keyboardManager, err := f.CreateKeyboardManager(api, whitelistManager, cache, workerPool)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyboard manager: %w", err)
	}

	// Создаем playlist service (для обратной совместимости)
	playlistService := playlist.NewPlaylistService(f.logger)

	// Создаем playlist manager (новый способ управления плейлистами)
	playlistManager := playlist.NewManager(f.logger, f.config.GetAppDataDir())

	// Загружаем плейлист из CSV файла только если указан путь
	if f.config.PlaylistCSVPath != "" {
		if err := playlistManager.LoadPlaylistFromFile(f.config.PlaylistCSVPath); err != nil {
			f.logger.Warn("Failed to load playlist from config path", zap.Error(err))
		} else {
			f.logger.Info("Playlist loaded from config path", zap.String("path", f.config.PlaylistCSVPath))
		}
	} else {
		// Пытаемся загрузить из постоянного хранилища
		if err := playlistManager.LoadPlaylistFromStorage(); err != nil {
			f.logger.Info("No playlist found in storage - playlist will be loaded via /import_playlist command")
		} else {
			f.logger.Info("Playlist loaded from storage")
		}
	}

	// Создаем кэш домашних заданий
	homeworkCache := playlist.NewHomeworkCache()

	deps := &types.Dependencies{
		BotAPI:          api,
		Logger:          f.logger,
		Config:          f.config,
		ReleaseService:  releaseService,
		ArtistService:   artistService,
		Keyboard:        keyboardManager,
		Debouncer:       debouncer,
		Cache:           cache,
		WorkerPool:      workerPool,
		PlaylistService: playlistService,
		PlaylistManager: playlistManager,
		HomeworkCache:   homeworkCache,
		Metrics:         metrics,
	}

	f.logger.Info("All dependencies created successfully")
	return deps, nil
}

// ValidateConfig проверяет конфигурацию на корректность
func (f *ComponentFactory) ValidateConfig() error {
	if f.config == nil {
		return fmt.Errorf("config is nil")
	}

	return f.config.Validate()
}
