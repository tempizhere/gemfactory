-- Create database and main tables
-- Migration: 001_init_clean.sql

-- Create release types table
CREATE TABLE IF NOT EXISTS release_types (
    release_type_id SERIAL PRIMARY KEY,
    name VARCHAR(20) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create artists table
CREATE TABLE IF NOT EXISTS artists (
    artist_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    gender VARCHAR(10) NOT NULL CHECK (gender IN ('female', 'male', 'mixed')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create releases table
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

-- Create homework table
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

-- Create playlists table
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

-- Create config table
CREATE TABLE IF NOT EXISTS config (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for query optimization
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

-- Create triggers for automatic updated_at update
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

-- Insert initial release types data
INSERT INTO release_types (name, description) VALUES
('single', 'Single'),
('album', 'Album'),
('ep', 'EP (Extended Play)')
ON CONFLICT (name) DO NOTHING;

-- Insert initial artists data
INSERT INTO artists (name, gender) VALUES
-- Female groups
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
-- Male groups
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

-- Insert default configuration
INSERT INTO config (key, value, description) VALUES
('RATE_LIMIT_REQUESTS', '10', 'Maximum requests in rate limiting window'),
('RATE_LIMIT_WINDOW', '60', 'Rate limiting window in seconds'),
('SCRAPER_DELAY', '1', 'Scraper delay between requests in seconds'),
('SCRAPER_TIMEOUT', '30', 'Scraper request timeout in seconds'),
('LOG_LEVEL', 'info', 'Logging level (debug, info, warn, error, fatal)'),
('PLAYLIST_UPDATE_HOURS', '24', 'Playlist update interval in hours'),
('BOT_TOKEN', '', 'Telegram bot token'),
('ADMIN_USERNAME', '', 'Administrator username'),
('SPOTIFY_CLIENT_ID', '', 'Spotify client ID'),
('SPOTIFY_CLIENT_SECRET', '', 'Spotify client secret'),
('PLAYLIST_URL', '', 'Spotify playlist URL'),
('DB_DSN', '', 'Database connection DSN'),
('HEALTH_PORT', '8080', 'Health check server port'),
('TIMEZONE', 'Europe/Moscow', 'Application timezone'),
('HEALTH_CHECK_ENABLED', 'true', 'Enable health check server'),
('APP_DATA_DIR', './data', 'Application data directory')
ON CONFLICT (key) DO NOTHING;
