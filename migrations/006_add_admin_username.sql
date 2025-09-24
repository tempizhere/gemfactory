-- Add admin username configuration if not exists
INSERT INTO config (key, value, description) VALUES
('ADMIN_USERNAME', 'admin', 'Administrator username for help command')
ON CONFLICT (key) DO NOTHING;
