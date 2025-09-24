-- migrations/005_init_next_run.sql

-- Инициализируем next_run для существующих задач
UPDATE tasks
SET next_run = CASE
    WHEN cron_expression = '0 2 * * *' THEN
        CASE
            WHEN EXTRACT(HOUR FROM NOW()) < 2 THEN
                DATE_TRUNC('day', NOW()) + INTERVAL '2 hours'
            ELSE
                DATE_TRUNC('day', NOW()) + INTERVAL '1 day' + INTERVAL '2 hours'
        END
    WHEN cron_expression = '0 3 */3 * *' THEN
        CASE
            WHEN EXTRACT(HOUR FROM NOW()) < 3 THEN
                DATE_TRUNC('day', NOW()) + INTERVAL '3 hours'
            ELSE
                DATE_TRUNC('day', NOW()) + INTERVAL '3 days' + INTERVAL '3 hours'
        END
    WHEN cron_expression = '0 6 * * *' THEN
        CASE
            WHEN EXTRACT(HOUR FROM NOW()) < 6 THEN
                DATE_TRUNC('day', NOW()) + INTERVAL '6 hours'
            ELSE
                DATE_TRUNC('day', NOW()) + INTERVAL '1 day' + INTERVAL '6 hours'
        END
    WHEN cron_expression = '0 */6 * * *' THEN
        CASE
            WHEN EXTRACT(HOUR FROM NOW()) < 6 THEN
                DATE_TRUNC('day', NOW()) + INTERVAL '6 hours'
            WHEN EXTRACT(HOUR FROM NOW()) < 12 THEN
                DATE_TRUNC('day', NOW()) + INTERVAL '12 hours'
            WHEN EXTRACT(HOUR FROM NOW()) < 18 THEN
                DATE_TRUNC('day', NOW()) + INTERVAL '18 hours'
            ELSE
                DATE_TRUNC('day', NOW()) + INTERVAL '1 day' + INTERVAL '6 hours'
        END
    ELSE
        NOW() + INTERVAL '1 hour'
END
WHERE next_run IS NULL;
