#!/bin/sh

# Проверяем, существует ли директория WHITELIST_DIR
WHITELIST_DIR=${WHITELIST_DIR:-/app/data}

# Создаем директорию, если она не существует
mkdir -p "$WHITELIST_DIR" || {
    echo "Error: Failed to create directory $WHITELIST_DIR"
    exit 1
}

# Проверяем наличие исходной директории
SOURCE_DIR="/app/internal/telegrambot/releases/data"

# Инициализируем female_whitelist.json
if [ ! -f "$WHITELIST_DIR/female_whitelist.json" ]; then
    echo "File female_whitelist.json not found in $WHITELIST_DIR. Initializing..."
    if [ -f "$SOURCE_DIR/female_whitelist.json" ]; then
        cp "$SOURCE_DIR/female_whitelist.json" "$WHITELIST_DIR/" || {
            echo "Error: Failed to copy female_whitelist.json"
            exit 1
        }
    else
        echo "Warning: Source file $SOURCE_DIR/female_whitelist.json not found. Creating empty file."
        echo "[]" > "$WHITELIST_DIR/female_whitelist.json" || {
            echo "Error: Failed to create female_whitelist.json"
            exit 1
        }
    fi
    chmod u+rw "$WHITELIST_DIR/female_whitelist.json" || {
        echo "Warning: Cannot set write permissions for female_whitelist.json"
    }
else
    echo "File female_whitelist.json already exists. Ensuring write permissions..."
    chmod u+rw "$WHITELIST_DIR/female_whitelist.json" 2>/dev/null || echo "Warning: Cannot set write permissions"
fi

# Инициализируем male_whitelist.json
if [ ! -f "$WHITELIST_DIR/male_whitelist.json" ]; then
    echo "File male_whitelist.json not found in $WHITELIST_DIR. Initializing..."
    if [ -f "$SOURCE_DIR/male_whitelist.json" ]; then
        cp "$SOURCE_DIR/male_whitelist.json" "$WHITELIST_DIR/" || {
            echo "Error: Failed to copy male_whitelist.json"
            exit 1
        }
    else
        echo "Warning: Source file $SOURCE_DIR/male_whitelist.json not found. Creating empty file."
        echo "[]" > "$WHITELIST_DIR/male_whitelist.json" || {
            echo "Error: Failed to create male_whitelist.json"
            exit 1
        }
    fi
    chmod u+rw "$WHITELIST_DIR/male_whitelist.json" || {
        echo "Warning: Cannot set write permissions for male_whitelist.json"
    }
else
    echo "File male_whitelist.json already exists. Ensuring write permissions..."
    chmod u+rw "$WHITELIST_DIR/male_whitelist.json" 2>/dev/null || echo "Warning: Cannot set write permissions"
fi

# Запускаем приложение
exec ./gemfactory