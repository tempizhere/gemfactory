-- Откат инициализации базы данных
-- Migration: 001_init_complete.down.sql

-- Установка схемы по умолчанию
SET search_path TO gemfactory, public;

-- Удаление таблиц в обратном порядке (из-за внешних ключей)
DROP TABLE IF EXISTS gemfactory.playlist_tracks CASCADE;
DROP TABLE IF EXISTS gemfactory.homeworks CASCADE;
DROP TABLE IF EXISTS gemfactory.tasks CASCADE;
DROP TABLE IF EXISTS gemfactory.config CASCADE;
DROP TABLE IF EXISTS gemfactory.releases CASCADE;
DROP TABLE IF EXISTS gemfactory.artists CASCADE;

-- Удаление схемы
DROP SCHEMA IF EXISTS gemfactory CASCADE;
