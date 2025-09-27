-- Полная инициализация базы данных
-- Migration: 001_init_complete.up.sql

-- Создание схемы gemfactory (если не существует)
CREATE SCHEMA IF NOT EXISTS gemfactory;

-- Установка схемы по умолчанию
SET search_path TO gemfactory, public;

-- Создание таблицы артистов
CREATE TABLE IF NOT EXISTS gemfactory.artists (
    artist_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    gender VARCHAR(10) NOT NULL CHECK (gender IN ('female', 'male', 'mixed')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы релизов
CREATE TABLE IF NOT EXISTS gemfactory.releases (
    release_id SERIAL PRIMARY KEY,
    artist_id INTEGER NOT NULL REFERENCES gemfactory.artists(artist_id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    title_track VARCHAR(500),
    album_name VARCHAR(500),
    mv VARCHAR(1000),
    date VARCHAR(20) NOT NULL,
    time_msk VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы конфигурации
CREATE TABLE IF NOT EXISTS gemfactory.config (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы задач
CREATE TABLE IF NOT EXISTS gemfactory.tasks (
    task_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    task_type VARCHAR(50) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    run_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы домашних заданий
CREATE TABLE IF NOT EXISTS gemfactory.homeworks (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    spotify_id VARCHAR(255) NOT NULL,
    play_count INTEGER NOT NULL DEFAULT 1,
    issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, track_id, spotify_id)
);

-- Создание таблицы треков плейлиста
CREATE TABLE IF NOT EXISTS gemfactory.playlist_tracks (
    id SERIAL PRIMARY KEY,
    spotify_id VARCHAR(255) NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    artist VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    album VARCHAR(500),
    duration_ms INTEGER,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(spotify_id, track_id)
);

-- Создание индексов для оптимизации
CREATE INDEX IF NOT EXISTS idx_releases_artist_id ON gemfactory.releases(artist_id);
CREATE INDEX IF NOT EXISTS idx_releases_date ON gemfactory.releases(date);
CREATE INDEX IF NOT EXISTS idx_releases_artist_date_track ON gemfactory.releases(artist_id, date, title_track);
CREATE INDEX IF NOT EXISTS idx_artists_name ON gemfactory.artists(name);
CREATE INDEX IF NOT EXISTS idx_artists_gender ON gemfactory.artists(gender);
CREATE INDEX IF NOT EXISTS idx_artists_active ON gemfactory.artists(is_active);
CREATE INDEX IF NOT EXISTS idx_homeworks_user_id ON gemfactory.homeworks(user_id);
CREATE INDEX IF NOT EXISTS idx_homeworks_track_id ON gemfactory.homeworks(track_id);
CREATE INDEX IF NOT EXISTS idx_homeworks_spotify_id ON gemfactory.homeworks(spotify_id);
CREATE INDEX IF NOT EXISTS idx_homeworks_completed ON gemfactory.homeworks(is_completed);
CREATE INDEX IF NOT EXISTS idx_homeworks_user_completed ON gemfactory.homeworks(user_id, is_completed);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_spotify_id ON gemfactory.playlist_tracks(spotify_id);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_track_id ON gemfactory.playlist_tracks(track_id);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_artist ON gemfactory.playlist_tracks(artist);
CREATE INDEX IF NOT EXISTS idx_tasks_name ON gemfactory.tasks(name);
CREATE INDEX IF NOT EXISTS idx_tasks_type ON gemfactory.tasks(task_type);
CREATE INDEX IF NOT EXISTS idx_tasks_active ON gemfactory.tasks(is_active);

-- Заполнение конфигурации по умолчанию
INSERT INTO gemfactory.config (key, value, description) VALUES
('RATE_LIMIT_REQUESTS', '10', 'Rate limit requests per window'),
('RATE_LIMIT_WINDOW', '60', 'Rate limit window in seconds'),
('SCRAPER_DELAY', '1', 'Scraper delay between requests in seconds'),
('SCRAPER_TIMEOUT', '30', 'Scraper request timeout in seconds'),
('LOG_LEVEL', 'info', 'Logging level (debug, info, warn, error, fatal)'),
('BOT_TOKEN', '', 'Telegram bot token'),
('ADMIN_USERNAME', '', 'Administrator username'),
('SPOTIFY_CLIENT_ID', '', 'Spotify client ID'),
('SPOTIFY_CLIENT_SECRET', '', 'Spotify client secret'),
('PLAYLIST_URL', '', 'Spotify playlist URL'),
('DB_DSN', '', 'Database connection string'),
('LLM_API_KEY', '', 'LLM API key'),
('LLM_BASE_URL', 'https://integrate.api.nvidia.com/v1', 'LLM base URL'),
('LLM_TIMEOUT', '120', 'LLM timeout in seconds'),
('LLM_DELAY', '1500', 'LLM request delay in milliseconds'),
('HEALTH_PORT', '8080', 'Health check port'),
('TIMEZONE', 'Europe/Moscow', 'Application timezone'),
('APP_DATA_DIR', './data', 'Application data directory')
ON CONFLICT (key) DO NOTHING;

-- Заполнение задач по умолчанию
INSERT INTO gemfactory.tasks (name, description, task_type, cron_expression, is_active, config) VALUES
('update_playlist_every_12h', 'Update playlist every 12 hours', 'update_playlist', '0 */12 * * *', TRUE, '{"description": "Update playlist every 12 hours"}'),
('homework_reset_daily', 'Reset homework assignments daily at midnight', 'homework_reset', '0 0 * * *', TRUE, '{"description": "Reset homework assignments daily at midnight"}'),
('parse_previous_months', 'Parse previous months of current year every 10 days', 'parse_releases', '0 3 */10 * *', TRUE, '{"description": "Parse previous months of current year every 10 days", "months": "previous_current_year"}'),
('parse_current_months', 'Parse current month and next 2 months daily', 'parse_releases', '0 2 * * *', TRUE, '{"description": "Parse current month and next 2 months daily", "months": "current+2"}')
ON CONFLICT (name) DO NOTHING;
