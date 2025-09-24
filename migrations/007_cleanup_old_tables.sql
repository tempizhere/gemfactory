-- Cleanup old tables
-- Migration: 007_cleanup_old_tables.sql

-- Drop old homework table (replaced by homeworks)
DROP TABLE IF EXISTS homework;
