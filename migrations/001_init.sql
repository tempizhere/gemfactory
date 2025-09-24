-- Создание базы данных и основных таблиц
-- Migration: 001_init.sql

-- Создание таблицы типов релизов
CREATE TABLE IF NOT EXISTS release_types (
    release_type_id SERIAL PRIMARY KEY,
    name VARCHAR(20) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы артистов
CREATE TABLE IF NOT EXISTS artists (
    artist_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    gender VARCHAR(10) NOT NULL CHECK (gender IN ('female', 'male', 'mixed')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы релизов
CREATE TABLE IF NOT EXISTS releases (
    release_id SERIAL PRIMARY KEY,
    artist_id INTEGER NOT NULL REFERENCES artists(artist_id) ON DELETE CASCADE,
    release_type_id INTEGER NOT NULL REFERENCES release_types(release_type_id) ON DELETE RESTRICT,
    title VARCHAR(500) NOT NULL,
    title_track VARCHAR(255),
    album_name VARCHAR(255),
    mv TEXT,
    date VARCHAR(50) NOT NULL,
    time_msk VARCHAR(10),
    month VARCHAR(20) NOT NULL,
    year INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы домашних заданий
CREATE TABLE IF NOT EXISTS homework (
    homework_id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    artist VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    play_count INTEGER NOT NULL DEFAULT 1,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы плейлистов
CREATE TABLE IF NOT EXISTS playlists (
    id SERIAL PRIMARY KEY,
    spotify_id VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(500) NOT NULL,
    description TEXT,
    owner VARCHAR(255) NOT NULL,
    track_count INTEGER NOT NULL DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы конфигурации
CREATE TABLE IF NOT EXISTS config (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание индексов для оптимизации запросов
CREATE INDEX IF NOT EXISTS idx_artists_name ON artists(name);
CREATE INDEX IF NOT EXISTS idx_artists_gender ON artists(gender);
CREATE INDEX IF NOT EXISTS idx_artists_is_active ON artists(is_active);
CREATE INDEX IF NOT EXISTS idx_artists_gender_active ON artists(gender, is_active);

CREATE INDEX IF NOT EXISTS idx_releases_artist_id ON releases(artist_id);
CREATE INDEX IF NOT EXISTS idx_releases_type_id ON releases(release_type_id);
CREATE INDEX IF NOT EXISTS idx_releases_month ON releases(month);
CREATE INDEX IF NOT EXISTS idx_releases_year ON releases(year);
CREATE INDEX IF NOT EXISTS idx_releases_is_active ON releases(is_active);
CREATE INDEX IF NOT EXISTS idx_releases_album_name ON releases(album_name);
CREATE INDEX IF NOT EXISTS idx_releases_title_track ON releases(title_track);
CREATE INDEX IF NOT EXISTS idx_releases_time_msk ON releases(time_msk);
CREATE INDEX IF NOT EXISTS idx_releases_type_gender ON releases(release_type_id, artist_id);
CREATE INDEX IF NOT EXISTS idx_releases_month_year ON releases(month, year);

CREATE INDEX IF NOT EXISTS idx_homework_user_id ON homework(user_id);
CREATE INDEX IF NOT EXISTS idx_homework_completed ON homework(completed);

CREATE INDEX IF NOT EXISTS idx_playlists_spotify_id ON playlists(spotify_id);

CREATE INDEX IF NOT EXISTS idx_config_key ON config(key);

-- Создание триггеров для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_release_types_updated_at BEFORE UPDATE ON release_types
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_artists_updated_at BEFORE UPDATE ON artists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_releases_updated_at BEFORE UPDATE ON releases
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_homework_updated_at BEFORE UPDATE ON homework
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_playlists_updated_at BEFORE UPDATE ON playlists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_config_updated_at BEFORE UPDATE ON config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Добавление начальных данных типов релизов
INSERT INTO release_types (name, description) VALUES
('single', 'Сингл'),
('album', 'Альбом'),
('ep', 'EP (Extended Play)')
ON CONFLICT (name) DO NOTHING;

-- Добавление начальных данных артистов
INSERT INTO artists (name, gender) VALUES
-- Женские группы
('aespa', 'female'),
('blackpink', 'female'),
('twice', 'female'),
('red velvet', 'female'),
('itzy', 'female'),
('newjeans', 'female'),
('ive', 'female'),
('le sserafim', 'female'),
('gidle', 'female'),
('stayc', 'female'),
('nmixx', 'female'),
('kep1er', 'female'),
('loona', 'female'),
('dreamcatcher', 'female'),
('mamamoo', 'female'),
('gfriend', 'female'),
('apink', 'female'),
('girls generation', 'female'),
('2ne1', 'female'),
('wonder girls', 'female'),
-- Мужские группы
('bts', 'male'),
('seventeen', 'male'),
('stray kids', 'male'),
('txt', 'male'),
('enhypen', 'male'),
('nct', 'male'),
('exo', 'male'),
('shinee', 'male'),
('super junior', 'male'),
('bigbang', 'male'),
('got7', 'male'),
('monsta x', 'male'),
('ateez', 'male'),
('the boyz', 'male'),
('treasure', 'male'),
('ikon', 'male'),
('winner', 'male'),
('block b', 'male'),
('infinite', 'male')
ON CONFLICT (name) DO NOTHING;

-- Добавление конфигурации по умолчанию
INSERT INTO config (key, value, description) VALUES
('RATE_LIMIT_REQUESTS', '10', 'Максимальное количество запросов в окне rate limiting'),
('RATE_LIMIT_WINDOW', '60', 'Окно rate limiting в секундах'),
('SCRAPER_DELAY', '1', 'Задержка между запросами скрапера в секундах'),
('SCRAPER_TIMEOUT', '30', 'Таймаут запросов скрапера в секундах'),
('LOG_LEVEL', 'info', 'Уровень логирования (debug, info, warn, error, fatal)'),
('PLAYLIST_UPDATE_HOURS', '24', 'Интервал обновления плейлиста в часах'),
('BOT_TOKEN', '', 'Токен Telegram бота'),
('ADMIN_USERNAME', '', 'Имя пользователя администратора'),
('SPOTIFY_CLIENT_ID', '', 'ID клиента Spotify'),
('SPOTIFY_CLIENT_SECRET', '', 'Секрет клиента Spotify'),
('PLAYLIST_URL', '', 'URL плейлиста Spotify'),
('DB_DSN', '', 'DSN подключения к базе данных'),
('HEALTH_PORT', '8080', 'Порт для health check сервера'),
('TIMEZONE', 'Europe/Moscow', 'Часовой пояс приложения'),
('HEALTH_CHECK_ENABLED', 'true', 'Включить health check сервер'),
('APP_DATA_DIR', './data', 'Директория для данных приложения')
ON CONFLICT (key) DO NOTHING;