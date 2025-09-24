-- Fix playlist structure for homework functionality
-- Migration: 004_fix_playlist_structure.sql

-- Rename playlists table to playlist_info (for metadata)
ALTER TABLE playlists RENAME TO playlist_info;

-- Create playlist table for storing tracks
CREATE TABLE IF NOT EXISTS playlist (
    id SERIAL PRIMARY KEY,
    spotify_id VARCHAR(255) NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    artist VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    album VARCHAR(255),
    duration_ms INTEGER,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(spotify_id, track_id)
);

-- Create homework_tracking table to track issued homework
CREATE TABLE IF NOT EXISTS homework_tracking (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    spotify_id VARCHAR(255) NOT NULL,
    issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, track_id, spotify_id)
);

-- Create indexes for optimization
CREATE INDEX IF NOT EXISTS idx_playlist_spotify_id ON playlist(spotify_id);
CREATE INDEX IF NOT EXISTS idx_playlist_track_id ON playlist(track_id);
CREATE INDEX IF NOT EXISTS idx_playlist_artist ON playlist(artist);

CREATE INDEX IF NOT EXISTS idx_homework_tracking_user_id ON homework_tracking(user_id);
CREATE INDEX IF NOT EXISTS idx_homework_tracking_track_id ON homework_tracking(track_id);
CREATE INDEX IF NOT EXISTS idx_homework_tracking_spotify_id ON homework_tracking(spotify_id);
CREATE INDEX IF NOT EXISTS idx_homework_tracking_completed ON homework_tracking(is_completed);
CREATE INDEX IF NOT EXISTS idx_homework_tracking_user_completed ON homework_tracking(user_id, is_completed);
