-- Migration: Add tasks table for scheduling parsing jobs
-- Description: Creates tasks table for managing scheduled parsing operations

-- Create tasks table
CREATE TABLE tasks (
    task_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    task_type VARCHAR(50) NOT NULL, -- 'parse_releases', 'update_playlist', etc.
    cron_expression VARCHAR(100) NOT NULL, -- Cron expression for scheduling
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    run_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    config JSONB, -- Additional configuration for the task
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_tasks_type ON tasks(task_type);
CREATE INDEX idx_tasks_active ON tasks(is_active);
CREATE INDEX idx_tasks_next_run ON tasks(next_run);

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_tasks_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_tasks_updated_at();

-- Insert default parsing tasks
INSERT INTO tasks (name, description, task_type, cron_expression, config) VALUES
-- Parse current month + 2 next months daily
('parse_current_months', 'Parse current month and next 2 months daily', 'parse_releases', '0 2 * * *', '{"months": "current+2", "description": "Parse current month and next 2 months"}'),

-- Parse previous months of current year every 3 days
('parse_previous_months', 'Parse previous months of current year every 3 days', 'parse_releases', '0 3 */3 * *', '{"months": "previous_current_year", "description": "Parse previous months of current year"}');

-- Update sequence
SELECT setval('tasks_task_id_seq', (SELECT MAX(task_id) FROM tasks));
