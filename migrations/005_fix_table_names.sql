-- Fix table names and structure
-- Migration: 005_fix_table_names.sql

-- Drop the unnecessary playlist_info table
DROP TABLE IF EXISTS playlist_info;

-- Rename homework_tracking to homeworks
ALTER TABLE homework_tracking RENAME TO homeworks;

-- Rename playlist to playlist_tracks
ALTER TABLE playlist RENAME TO playlist_tracks;

-- Update indexes
DROP INDEX IF EXISTS idx_homework_tracking_user_id;
DROP INDEX IF EXISTS idx_homework_tracking_track_id;
DROP INDEX IF EXISTS idx_homework_tracking_spotify_id;
DROP INDEX IF EXISTS idx_homework_tracking_completed;
DROP INDEX IF EXISTS idx_homework_tracking_user_completed;

DROP INDEX IF EXISTS idx_playlist_spotify_id;
DROP INDEX IF EXISTS idx_playlist_track_id;
DROP INDEX IF EXISTS idx_playlist_artist;

-- Create new indexes with correct names
CREATE INDEX IF NOT EXISTS idx_homeworks_user_id ON homeworks(user_id);
CREATE INDEX IF NOT EXISTS idx_homeworks_track_id ON homeworks(track_id);
CREATE INDEX IF NOT EXISTS idx_homeworks_spotify_id ON homeworks(spotify_id);
CREATE INDEX IF NOT EXISTS idx_homeworks_completed ON homeworks(is_completed);
CREATE INDEX IF NOT EXISTS idx_homeworks_user_completed ON homeworks(user_id, is_completed);

CREATE INDEX IF NOT EXISTS idx_playlist_tracks_spotify_id ON playlist_tracks(spotify_id);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_track_id ON playlist_tracks(track_id);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_artist ON playlist_tracks(artist);
