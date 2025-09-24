-- Add homework reset configuration
-- Migration: 008_add_homework_reset_config.sql

-- Add homework reset time configuration
INSERT INTO config (key, value, description)
VALUES ('HOMEWORK_RESET_TIME', '00:00', 'Время сброса домашних заданий (формат HH:MM)')
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP;
