// Package main запускает Telegram-бота GemFactory.
package main

import (
	"context"
	"gemfactory/internal/telegrambot/bot"
	"gemfactory/pkg/config"
	"gemfactory/pkg/log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// Инициализация логгера
	logger, err := log.Init()
	if err != nil {
		// Используем os.Exit вместо panic для корректного завершения
		if _, err2 := os.Stderr.WriteString("Failed to initialize logger: " + err.Error() + "\n"); err2 != nil {
			panic("Failed to write to stderr: " + err2.Error())
		}
		os.Exit(1)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			_, _ = os.Stderr.WriteString("Failed to sync logger: " + err.Error() + "\n")
		}
	}()

	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Создание бота
	b, err := bot.NewBot(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create bot", zap.Error(err))
	}

	// Канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запуск бота в горутине
	go func() {
		if err := b.Start(); err != nil {
			logger.Error("Bot stopped with error", zap.Error(err))
		}
	}()

	// Ожидание сигнала завершения
	<-sigChan
	logger.Info("Received shutdown signal, starting graceful shutdown")

	// Создание контекста с таймаутом для graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Запуск graceful shutdown в горутине
	go func() {
		if err := b.Stop(); err != nil {
			logger.Error("Graceful shutdown failed", zap.Error(err))
		}
	}()

	// Ожидание завершения или таймаута
	select {
	case <-shutdownCtx.Done():
		logger.Warn("Graceful shutdown timeout exceeded", zap.Error(shutdownCtx.Err()))
		logger.Error("Graceful shutdown failed", zap.Error(shutdownCtx.Err()))
		return
	default:
		logger.Info("Graceful shutdown completed successfully")
	}
}
