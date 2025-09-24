#!/bin/bash

# Тестовый скрипт для проверки Docker конфигурации
set -e

echo "Тестирование Docker конфигурации для внешней базы данных..."

# Проверяем наличие файлов
echo "Проверка файлов:"
if [ -f "deployments/docker-compose.yml" ]; then
    echo "✓ docker-compose.yml найден"
else
    echo "✗ docker-compose.yml не найден"
    exit 1
fi

if [ -f "deployments/Dockerfile" ]; then
    echo "✓ Dockerfile найден"
else
    echo "✗ Dockerfile не найден"
    exit 1
fi

if [ -f "deployments/start.sh" ]; then
    echo "✓ start.sh найден"
else
    echo "✗ start.sh не найден"
    exit 1
fi

if [ -f "scripts/deploy.sh" ]; then
    echo "✓ deploy.sh найден"
else
    echo "✗ deploy.sh не найден"
    exit 1
fi

# Проверяем наличие миграций
if [ -d "migrations" ]; then
    echo "✓ Папка migrations найдена"
    echo "  Найдено миграций: $(ls migrations/*.sql 2>/dev/null | wc -l)"
else
    echo "✗ Папка migrations не найдена"
    exit 1
fi

# Проверяем .env файл
if [ -f ".env" ]; then
    echo "✓ .env файл найден"
    source .env
    if [ -n "$DB_DSN" ]; then
        echo "✓ DB_DSN установлена"
    else
        echo "✗ DB_DSN не установлена"
    fi
    if [ -n "$BOT_TOKEN" ]; then
        echo "✓ BOT_TOKEN установлен"
    else
        echo "✗ BOT_TOKEN не установлен"
    fi
else
    echo "⚠ .env файл не найден (создайте из .env.example)"
fi

echo ""
echo "Конфигурация готова для развертывания!"
echo ""
echo "Для запуска используйте:"
echo "  ./scripts/deploy.sh up"
echo ""
echo "Для просмотра логов:"
echo "  ./scripts/deploy.sh logs"



