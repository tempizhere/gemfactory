package main

import (
	"fmt"

	"gemfactory/bot"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load environment variables (optional)
	_ = godotenv.Load() // Игнорируем ошибку, переменные будут браться из окружения

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}
	defer logger.Sync()

	// Load configuration
	config, err := bot.NewConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
		return
	}

	// Initialize bot
	botInstance, err := bot.NewBot(config, logger)
	if err != nil {
		logger.Fatal("Failed to initialize bot", zap.Error(err))
		return
	}

	// Start bot
	if err := botInstance.Start(); err != nil {
		logger.Fatal("Failed to start bot", zap.Error(err))
	}
}
