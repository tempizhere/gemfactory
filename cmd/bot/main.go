// Package main запускает Telegram-бота GemFactory.
package main

import (
	"context"
	"gemfactory/internal/app"
	"gemfactory/internal/config"
	"gemfactory/pkg/logger"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	// Инициализация логгера
	log := logger.New()

	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Создание контекста
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Shutdown signal received")
		cancel()
	}()

	// Создание и запуск бота через фабрику
	bot, err := app.NewBotWithFactory(cfg, log)
	if err != nil {
		log.Fatal("Failed to create bot", zap.Error(err))
	}

	// Запуск бота
	if err := bot.Start(ctx); err != nil {
		log.Error("Bot stopped with error", zap.Error(err))
		os.Exit(1)
	}

	log.Info("Bot stopped successfully")
}
