// Package main запускает Telegram-бота GemFactory.
package main

import (
	"context"
	"fmt"
	"gemfactory/internal/bot"
	"gemfactory/internal/config"
	"gemfactory/pkg/log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// formatDuration форматирует время в читаемый формат (например: 8s)
func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	return fmt.Sprintf("%ds", seconds)
}

// AppInfo содержит информацию о приложении
type AppInfo struct {
	Name      string
	Version   string
	BuildTime string
	GitCommit string
}

// getAppInfo возвращает информацию о приложении из конфигурации
func getAppInfo(cfg *config.Config) AppInfo {
	return AppInfo{
		Name:      cfg.GetAppName(),
		Version:   cfg.GetAppVersion(),
		BuildTime: cfg.GetBuildTime(),
		GitCommit: cfg.GetGitCommit(),
	}
}

// gracefulShutdown выполняет graceful shutdown с таймаутом
func gracefulShutdown(ctx context.Context, logger *zap.Logger, bot interface{ Stop() error }, cfg *config.Config, wg *sync.WaitGroup) {
	defer wg.Done()

	<-ctx.Done()
	logger.Info("Shutdown signal received, starting graceful shutdown")

	// Создаем контекст с таймаутом для shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GetGracefulShutdownTimeout())
	defer cancel()

	// Канал для получения результата shutdown
	done := make(chan error, 1)

	go func() {
		done <- bot.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error("Error during graceful shutdown", zap.Error(err))
		} else {
			logger.Info("Graceful shutdown completed successfully")
		}
	case <-shutdownCtx.Done():
		logger.Error("Graceful shutdown timeout exceeded, forcing exit")
	}
}

// setupSignalHandler настраивает обработку сигналов
func setupSignalHandler(ctx context.Context, cancel context.CancelFunc, logger *zap.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		select {
		case sig := <-sigChan:
			logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
			cancel()
		case <-ctx.Done():
			return
		}
	}()
}

// logSystemInfo логирует информацию о системе
func logSystemInfo(logger *zap.Logger, appInfo AppInfo) {
	logger.Info("Starting application",
		zap.String("app_name", appInfo.Name),
		zap.String("version", appInfo.Version),
		zap.String("build_time", appInfo.BuildTime),
		zap.String("git_commit", appInfo.GitCommit),
		zap.String("go_version", runtime.Version()),
		zap.String("go_os", runtime.GOOS),
		zap.String("go_arch", runtime.GOARCH),
		zap.Int("num_cpu", runtime.NumCPU()),
		zap.Int("num_goroutine", runtime.NumGoroutine()),
	)
}

func main() {
	// Инициализация
	startTime := time.Now()

	var logger *zap.Logger
	var err error

	// Инициализация логгера с обработкой ошибок
	logger, err = log.Init()
	if err != nil {
		// Критическая ошибка - используем stderr и завершаем приложение
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Настраиваем правильное завершение логгера
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			// Логгер может быть недоступен, используем stderr
			fmt.Fprintf(os.Stderr, "WARNING: Failed to sync logger: %v\n", syncErr)
		}
	}()

	// Загружаем и валидируем конфигурацию
	logger.Info("Loading configuration")
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.Error(err),
			zap.String("action", "check environment variables and config files"))
	}

	logger.Info("Configuration loaded successfully")

	// Логируем информацию о приложении и системе
	appInfo := getAppInfo(cfg)
	logSystemInfo(logger, appInfo)

	// Проверяем переменные окружения
	if err := cfg.ValidateEnvironment(); err != nil {
		logger.Fatal("Environment validation failed", zap.Error(err))
	}

	// Создаем контекст для управления жизненным циклом приложения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем обработку сигналов
	setupSignalHandler(ctx, cancel, logger)

	// Создаем бота
	logger.Info("Creating bot instance")
	b, err := bot.NewBot(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create bot",
			zap.Error(err),
			zap.String("action", "check bot token and network connectivity"))
	}

	logger.Info("Bot instance created successfully")

	// Логируем общее время инициализации
	initializationTime := time.Since(startTime)
	logger.Info("Application initialization completed",
		zap.String("total_initialization_time", formatDuration(initializationTime)))

	// WaitGroup для graceful shutdown
	var wg sync.WaitGroup

	// Запускаем graceful shutdown в отдельной горутине
	wg.Add(1)
	go gracefulShutdown(ctx, logger, b, cfg, &wg)

	// Запускаем бота
	logger.Info("Starting bot")
	if err := b.Start(); err != nil {
		// Проверяем, была ли ошибка вызвана отменой контекста
		if ctx.Err() != nil {
			logger.Info("Bot stopped due to context cancellation")
		} else {
			logger.Error("Bot stopped with error",
				zap.Error(err),
				zap.String("action", "check network connectivity and bot configuration"))
		}
	}

	// Ждем завершения graceful shutdown
	logger.Info("Waiting for graceful shutdown to complete")
	wg.Wait()

	// Финальное сообщение
	logger.Info("Application terminated successfully",
		zap.String("app_name", appInfo.Name))
}
