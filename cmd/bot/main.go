package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"gemfactory/internal/telegrambot/bot"
	"gemfactory/pkg/config"
	"gemfactory/pkg/log"
)

func main() {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
	}

	// Собираем список ключей переменных окружения (без значений)
	var envKeys []string
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "BOT_TOKEN=") ||
			strings.HasPrefix(env, "ADMIN_USERNAME=") ||
			strings.HasPrefix(env, "REQUEST_DELAY=") ||
			strings.HasPrefix(env, "MAX_RETRIES=") ||
			strings.HasPrefix(env, "CACHE_DURATION=") ||
			strings.HasPrefix(env, "WHITELIST_DIR=") ||
			strings.HasPrefix(env, "LOG_LEVEL=") {
			key := strings.SplitN(env, "=", 2)[0]
			envKeys = append(envKeys, key)
		}
	}
	// Выводим список ключей через запятую
	if len(envKeys) > 0 {
		fmt.Fprintf(os.Stderr, "Environment variables loaded: %s\n", strings.Join(envKeys, ","))
	} else {
		fmt.Fprintf(os.Stderr, "No environment variables loaded\n")
	}

	// Initialize logger
	logger, err := log.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logger = logger.WithOptions(zap.AddCallerSkip(1))
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", zap.Error(err))
		os.Exit(1)
	}

	// Initialize bot
	b, err := bot.NewBot(cfg, logger)
	if err != nil {
		logger.Error("Failed to initialize bot", zap.Error(err))
		os.Exit(1)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		os.Exit(0)
	}()

	// Start bot
	if err := b.Start(); err != nil {
		logger.Error("Bot stopped with error", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Bot stopped gracefully")
}
