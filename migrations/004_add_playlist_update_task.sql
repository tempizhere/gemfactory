-- migrations/004_add_playlist_update_task.sql

-- Добавление задачи обновления плейлистов
INSERT INTO tasks (name, description, task_type, cron_expression, is_active, config) VALUES
('update_playlist_daily', 'Update playlist daily at 6:00 AM', 'update_playlist', '0 6 * * *', TRUE, '{"description": "Daily playlist update at 6:00 AM"}')
ON CONFLICT (name) DO NOTHING;

-- Добавление задачи обновления плейлистов каждые 6 часов
INSERT INTO tasks (name, description, task_type, cron_expression, is_active, config) VALUES
('update_playlist_every_6h', 'Update playlist every 6 hours', 'update_playlist', '0 */6 * * *', TRUE, '{"description": "Update playlist every 6 hours"}')
ON CONFLICT (name) DO NOTHING;
