-- Add homework reset task
-- Migration: 009_add_homework_reset_task.sql

-- Add homework reset task
INSERT INTO tasks (name, description, task_type, cron_expression, is_active, created_at, updated_at)
VALUES (
    'homework_reset',
    'Автоматический сброс домашних заданий в указанное время',
    'homework_reset',
    '0 0 * * *', -- По умолчанию в полночь, будет обновлено из конфигурации
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    cron_expression = EXCLUDED.cron_expression,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;
