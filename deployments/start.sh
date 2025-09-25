#!/bin/sh
set -e

echo "=== Запуск GemFactory ==="

# Извлекаем параметры подключения из DB_DSN
DB_HOST=$(echo $DB_DSN | cut -d@ -f2 | cut -d: -f1)
DB_PORT=$(echo $DB_DSN | cut -d: -f4 | cut -d/ -f1)
DB_USER=$(echo $DB_DSN | cut -d/ -f3 | cut -d: -f1)
DB_PASSWORD=$(echo $DB_DSN | cut -d: -f3 | cut -d@ -f1)
DB_NAME=$(echo $DB_DSN | cut -d/ -f4 | cut -d? -f1)

echo "DB_HOST: $DB_HOST"
echo "DB_PORT: $DB_PORT"
echo "DB_USER: $DB_USER"
echo "DB_NAME: $DB_NAME"

# Ждем подключения к серверу PostgreSQL
echo "Ожидание подключения к серверу PostgreSQL..."
until pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER; do
  echo "Сервер PostgreSQL недоступен - ожидание..."
  sleep 2
done

# База данных и миграции уже применены через init-db.sql

# Проверяем, что таблицы созданы
echo "Проверка содержимого таблиц..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
SELECT 'config' as table_name, COUNT(*) as record_count FROM gemfactory.config
UNION ALL
SELECT 'tasks' as table_name, COUNT(*) as record_count FROM gemfactory.tasks
UNION ALL
SELECT 'artists' as table_name, COUNT(*) as record_count FROM gemfactory.artists;
"

# Запускаем приложение
echo "Запуск приложения..."
exec ./gemfactory