-- Add play_count field to homeworks table
-- Migration: 006_add_play_count_to_homeworks.sql

-- Add play_count column to homeworks table
ALTER TABLE homeworks ADD COLUMN IF NOT EXISTS play_count INTEGER NOT NULL DEFAULT 1;

-- Update existing records to have a random play_count between 1-6
UPDATE homeworks SET play_count = (RANDOM() * 5 + 1)::INTEGER WHERE play_count = 1;
