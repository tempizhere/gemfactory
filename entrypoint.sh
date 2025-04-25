#!/bin/sh

# Проверяем, существует ли директория WHITELIST_DIR в томе
WHITELIST_DIR=${WHITELIST_DIR:-internal/features/releasesbot/data}

if [ ! -f "$WHITELIST_DIR/female_whitelist.json" ] || [ ! -f "$WHITELIST_DIR/male_whitelist.json" ]; then
    echo "Initializing whitelist files in $WHITELIST_DIR"
    mkdir -p "$WHITELIST_DIR"
    cp /app/internal/features/releasesbot/data/female_whitelist.json "$WHITELIST_DIR/"
    cp /app/internal/features/releasesbot/data/male_whitelist.json "$WHITELIST_DIR/"
fi

# Запускаем приложение
exec ./gemfactory