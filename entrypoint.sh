#!/bin/sh

# Проверяем, существует ли директория WHITELIST_DIR в томе
WHITELIST_DIR=${WHITELIST_DIR:-/app/internal/features/releasesbot/data}

# Создаем директорию, если она не существует
mkdir -p "$WHITELIST_DIR"

# Устанавливаем правильного владельца для директории и файлов
chown -R appuser:appgroup "$WHITELIST_DIR"

# Проверяем наличие файлов и копируем их, если они отсутствуют
if [ ! -f "$WHITELIST_DIR/female_whitelist.json" ]; then
    echo "File female_whitelist.json not found. Initializing..."
    cp /app/internal/features/releasesbot/data/female_whitelist.json "$WHITELIST_DIR/"
else
    echo "File female_whitelist.json already exists. Skipping initialization."
fi

if [ ! -f "$WHITELIST_DIR/male_whitelist.json" ]; then
    echo "File male_whitelist.json not found. Initializing..."
    cp /app/internal/features/releasesbot/data/male_whitelist.json "$WHITELIST_DIR/"
else
    echo "File male_whitelist.json already exists. Skipping initialization."
fi

# Запускаем приложение
exec ./gemfactory