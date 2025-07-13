package bot

import (
	"gemfactory/internal/debounce"
	"gemfactory/internal/telegrambot/bot/botapi"
	commandcache "gemfactory/internal/telegrambot/bot/cache"
	"gemfactory/internal/telegrambot/bot/health"
	"gemfactory/internal/telegrambot/bot/keyboard"
	"gemfactory/internal/telegrambot/bot/metrics"
	"gemfactory/internal/telegrambot/bot/middleware"
	"gemfactory/internal/telegrambot/bot/service"
	"gemfactory/internal/telegrambot/bot/types"
	"gemfactory/internal/telegrambot/bot/worker"
	"gemfactory/internal/telegrambot/releases/artist"
	releasecache "gemfactory/internal/telegrambot/releases/cache"
	"gemfactory/internal/telegrambot/releases/scraper"
	"gemfactory/internal/telegrambot/releases/updater"
	"gemfactory/pkg/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// ComponentFactory создает компоненты бота
type ComponentFactory struct {
	config config.Interface
	logger *zap.Logger
}

// NewComponentFactory создает новую фабрику компонентов
func NewComponentFactory(config *config.Config, logger *zap.Logger) *ComponentFactory {
	return &ComponentFactory{
		config: config,
		logger: logger,
	}
}

// CreateBotAPI создает API для работы с Telegram
func (f *ComponentFactory) CreateBotAPI() (botapi.BotAPI, error) {
	tgAPI, err := tgbotapi.NewBotAPI(f.config.GetBotToken())
	if err != nil {
		return nil, err
	}

	api := botapi.NewTelegramBotAPI(tgAPI, f.logger)
	return api, nil
}

// CreateWhitelistManager создает менеджер белых списков
func (f *ComponentFactory) CreateWhitelistManager() artist.WhitelistManager {
	return artist.NewWhitelistManager(f.config.GetWhitelistDir(), f.logger)
}

// CreateScraper создает скрейпер
func (f *ComponentFactory) CreateScraper() scraper.Fetcher {
	// Приведение типа для совместимости с существующим кодом
	if cfg, ok := f.config.(*config.Config); ok {
		return scraper.NewFetcher(cfg, f.logger)
	}
	// Fallback для интерфейса
	return scraper.NewFetcher(nil, f.logger)
}

// CreateManager создает менеджер кэша
func (f *ComponentFactory) CreateManager(
	_ artist.WhitelistManager,
	scraper scraper.Fetcher,
	metrics metrics.Interface,
) releasecache.Cache {
	// Приведение типа для совместимости с существующим кодом
	if cfg, ok := f.config.(*config.Config); ok {
		whitelistManager := f.CreateWhitelistManager()
		manager := releasecache.NewManager(cfg, f.logger, whitelistManager, scraper, nil)
		updater := updater.NewUpdater(cfg, f.logger, whitelistManager, manager, scraper)
		manager.SetUpdater(updater)

		// Устанавливаем метрики в кэш и updater
		if metrics != nil {
			manager.SetMetrics(metrics)
			updater.SetMetrics(metrics)
		}

		return manager
	}
	// Fallback для интерфейса
	return nil
}

// CreateServices создает сервисы
func (f *ComponentFactory) CreateServices(
	_ artist.WhitelistManager,
	cache releasecache.Cache,
) (*service.ReleaseService, *service.ArtistService) {
	// Приведение типа для совместимости с существующим кодом
	if cfg, ok := f.config.(*config.Config); ok {
		whitelistManager := f.CreateWhitelistManager()
		releaseService := service.NewReleaseService(whitelistManager, cfg, f.logger, cache)
		artistService := service.NewArtistService(whitelistManager, f.logger)
		return releaseService, artistService
	}
	// Fallback для интерфейса
	return nil, nil
}

// CreateKeyboardManager создает менеджер клавиатуры
func (f *ComponentFactory) CreateKeyboardManager(
	api botapi.BotAPI,
	whitelistManager artist.WhitelistManager,
	cache releasecache.Cache,
) keyboard.ManagerInterface {
	// Приведение типа для совместимости с существующим кодом
	if cfg, ok := f.config.(*config.Config); ok {
		return keyboard.NewKeyboardManager(api, f.logger, whitelistManager, cfg, cache)
	}
	// Fallback для интерфейса
	return nil
}

// CreateWorkerPool создает пул воркеров
func (f *ComponentFactory) CreateWorkerPool() worker.PoolInterface {
	return worker.NewWorkerPool(f.config.GetMaxConcurrentRequests(), 100, f.logger)
}

// CreateCommandCache создает кэш команд
func (f *ComponentFactory) CreateCommandCache() commandcache.CommandCacheInterface {
	if !f.config.GetCommandCacheEnabled() {
		return nil
	}
	return commandcache.NewCommandCache(f.config.GetCommandCacheTTL(), f.logger)
}

// CreateRateLimiter создает ограничитель запросов
func (f *ComponentFactory) CreateRateLimiter() middleware.RateLimiterInterface {
	if !f.config.GetRateLimitEnabled() {
		return nil
	}
	return middleware.NewRateLimiter(
		f.config.GetRateLimitRequests(),
		f.config.GetRateLimitWindow(),
		f.logger,
	)
}

// CreateServer создает сервер health check
func (f *ComponentFactory) CreateServer() health.ServerInterface {
	if !f.config.GetHealthCheckEnabled() {
		return nil
	}
	return health.NewHealthServer(f.config.GetHealthCheckPort(), f.logger)
}

// CreateMetrics создает систему метрик
func (f *ComponentFactory) CreateMetrics() metrics.Interface {
	return metrics.NewMetrics(f.logger)
}

// CreateDependencies создает все зависимости
func (f *ComponentFactory) CreateDependencies(
	api botapi.BotAPI,
	_ artist.WhitelistManager,
	cache releasecache.Cache,
	releaseService *service.ReleaseService,
	artistService *service.ArtistService,
	keyboardManager keyboard.ManagerInterface,
	workerPool worker.PoolInterface,
	commandCache commandcache.CommandCacheInterface,
) *types.Dependencies {
	debouncer := debounce.NewDebouncer()

	// Создаем метрики
	metrics := f.CreateMetrics()

	return &types.Dependencies{
		BotAPI:         api,
		Logger:         f.logger,
		Config:         f.config,
		ReleaseService: releaseService,
		ArtistService:  artistService,
		Keyboard:       keyboardManager,
		Debouncer:      debouncer,
		Cache:          cache,
		WorkerPool:     workerPool,
		CommandCache:   commandCache,
		Metrics:        metrics,
	}
}
