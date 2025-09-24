-- Rollback homework reset configuration
DELETE FROM config WHERE key = 'HOMEWORK_RESET_TIME';
