package main

import (
    "context"

    "gemfactory/internal/features/releasesbot/bot"
    "gemfactory/internal/features/releasesbot/cache"
    "gemfactory/internal/features/releasesbot/artistlist"

    "gemfactory/pkg/log"
    "go.uber.org/zap"
)

func main() {
    // Initialize logger
    logger := log.Init()
    defer logger.Sync()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Load configuration
    config, err := bot.NewConfig()
    if err != nil {
        logger.Fatal("Failed to load configuration", zap.Error(err))
    }

    // Initialize artist list
    al, err := artistlist.NewArtistList(config.WhitelistDir, logger)
    if err != nil {
        logger.Fatal("Failed to initialize artist list", zap.Error(err))
    }

    // Запускаем автоматическое обновление кэша через cache/
    cache.StartUpdater(ctx, config, logger, al)

    // Initialize bot
    botInstance, err := bot.NewBot(config, logger)
    if err != nil {
        logger.Fatal("Failed to initialize bot", zap.Error(err))
    }

    // Start bot
    if err := botInstance.Start(); err != nil {
        logger.Fatal("Failed to start bot", zap.Error(err))
    }
}