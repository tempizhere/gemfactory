#!/bin/bash

# Скрипт для выполнения миграций базы данных
# Использование: ./scripts/migrate.sh [up|down|status]

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функция для вывода сообщений
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

# Проверяем наличие переменной окружения DB_DSN
if [ -z "$DB_DSN" ]; then
    error "DB_DSN не установлена"
    echo "Установите переменную окружения DB_DSN:"
    echo "export DB_DSN='postgres://user:password@localhost:5432/gemfactory?sslmode=disable'"
    exit 1
fi

# Проверяем наличие команды migrate
if ! command -v migrate &> /dev/null; then
    error "Команда migrate не найдена"
    echo "Установите golang-migrate:"
    echo "go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    exit 1
fi

# Определяем действие
ACTION=${1:-up}

case $ACTION in
    up)
        log "Выполнение миграций вверх..."
        migrate -path migrations -database "$DB_DSN" up
        log "Миграции выполнены успешно"
        ;;
    down)
        log "Выполнение миграций вниз..."
        migrate -path migrations -database "$DB_DSN" down
        log "Миграции откачены успешно"
        ;;
    status)
        log "Проверка статуса миграций..."
        migrate -path migrations -database "$DB_DSN" version
        ;;
    force)
        if [ -z "$2" ]; then
            error "Не указана версия для принудительной установки"
            echo "Использование: $0 force <version>"
            exit 1
        fi
        log "Принудительная установка версии $2..."
        migrate -path migrations -database "$DB_DSN" force "$2"
        log "Версия установлена успешно"
        ;;
    create)
        if [ -z "$2" ]; then
            error "Не указано имя миграции"
            echo "Использование: $0 create <name>"
            exit 1
        fi
        log "Создание новой миграции: $2..."
        migrate create -ext sql -dir migrations -seq "$2"
        log "Миграция создана успешно"
        ;;
    *)
        echo "Использование: $0 [up|down|status|force|create]"
        echo ""
        echo "Команды:"
        echo "  up      - Выполнить все миграции вверх"
        echo "  down    - Откатить последнюю миграцию"
        echo "  status  - Показать текущую версию миграций"
        echo "  force   - Принудительно установить версию"
        echo "  create  - Создать новую миграцию"
        echo ""
        echo "Примеры:"
        echo "  $0 up"
        echo "  $0 down"
        echo "  $0 status"
        echo "  $0 force 1"
        echo "  $0 create add_new_table"
        exit 1
        ;;
esac
