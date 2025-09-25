// Package storage содержит работу с базой данных.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"gemfactory/internal/model"
	"gemfactory/internal/storage/repository"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"go.uber.org/zap"
)

// Postgres представляет подключение к PostgreSQL
type Postgres struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewPostgres создает новое подключение к PostgreSQL с retry логикой
func NewPostgres(databaseURL string, logger *zap.Logger) (*Postgres, error) {
	const maxRetries = 10
	const retryDelay = 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Info("Attempting to connect to database",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries))

		// Создаем подключение к PostgreSQL
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseURL)))

		// Настраиваем пул соединений
		sqldb.SetMaxOpenConns(25)
		sqldb.SetMaxIdleConns(10)
		sqldb.SetConnMaxLifetime(5 * time.Minute)
		sqldb.SetConnMaxIdleTime(1 * time.Minute)

		// Создаем Bun DB
		db := bun.NewDB(sqldb, pgdialect.New())

		// Устанавливаем схему по умолчанию
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := db.ExecContext(ctx, "SET search_path TO gemfactory, public")
		cancel()
		if err != nil {
			logger.Warn("Failed to set search_path", zap.Error(err))
		}

		// Добавляем отладку в режиме разработки
		if logger.Core().Enabled(zap.DebugLevel) {
			db.AddQueryHook(bundebug.NewQueryHook(
				bundebug.WithVerbose(true),
				bundebug.FromEnv("BUNDEBUG"),
			))
		}

		// Проверяем подключение с таймаутом
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
		pingErr := db.PingContext(pingCtx)
		pingCancel()

		if pingErr != nil {
			logger.Warn("Failed to connect to database",
				zap.Int("attempt", attempt),
				zap.Error(pingErr))

			// Закрываем неудачное подключение
			if err := db.Close(); err != nil {
				logger.Warn("Failed to close database connection", zap.Error(err))
			}

			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
			}

			logger.Info("Retrying connection",
				zap.Duration("delay", retryDelay))
			time.Sleep(retryDelay)
			continue
		}

		logger.Info("Connected to PostgreSQL database with Bun ORM",
			zap.Int("attempt", attempt))

		return &Postgres{
			db:     db,
			logger: logger,
		}, nil
	}

	return nil, fmt.Errorf("unexpected error: max retries exceeded")
}

// Close закрывает соединение с базой данных
func (p *Postgres) Close() error {
	return p.db.Close()
}

// GetDB возвращает подключение к базе данных
func (p *Postgres) GetDB() *bun.DB {
	return p.db
}

// GetArtistRepository возвращает репозиторий артистов
func (p *Postgres) GetArtistRepository() model.ArtistRepository {
	return repository.NewArtistRepository(p.db, p.logger)
}

// GetReleaseRepository возвращает репозиторий релизов
func (p *Postgres) GetReleaseRepository() model.ReleaseRepository {
	return repository.NewReleaseRepository(p.db, p.logger)
}

// GetTaskRepository возвращает репозиторий задач
func (p *Postgres) GetTaskRepository() model.TaskRepository {
	return repository.NewTaskRepository(p.db, p.logger)
}

// GetHomeworkRepository возвращает репозиторий домашних заданий
func (p *Postgres) GetHomeworkRepository() model.HomeworkRepository {
	return repository.NewHomeworkRepository(p.db, p.logger)
}

// GetPlaylistTracksRepository возвращает репозиторий треков плейлиста
func (p *Postgres) GetPlaylistTracksRepository() model.PlaylistTracksRepository {
	return repository.NewPlaylistTracksRepository(p.db, p.logger)
}

// GetConfigRepository возвращает репозиторий конфигурации
func (p *Postgres) GetConfigRepository() model.ConfigRepository {
	return repository.NewConfigRepository(p.db, p.logger)
}
