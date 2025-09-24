#!/bin/bash

# Скрипт для развертывания приложения
# Использование: ./scripts/deploy.sh [local|docker|production]

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Определяем режим развертывания
DEPLOY_MODE=${1:-local}

# Проверяем наличие .env файла
if [ ! -f ".env" ]; then
    warning ".env файл не найден, создаем из примера..."
    if [ -f ".env.example" ]; then
        cp .env.example .env
        warning "Скопирован .env.example в .env"
        warning "Отредактируйте .env файл перед запуском"
    else
        error ".env файл не найден и .env.example недоступен"
        exit 1
    fi
fi

# Загружаем переменные окружения
source .env

case $DEPLOY_MODE in
    local)
        log "Локальное развертывание..."

        # Проверяем наличие PostgreSQL
        if ! command -v psql &> /dev/null; then
            error "PostgreSQL не установлен"
            echo "Установите PostgreSQL:"
            echo "  Ubuntu/Debian: sudo apt-get install postgresql postgresql-contrib"
            echo "  macOS: brew install postgresql"
            echo "  Windows: https://www.postgresql.org/download/windows/"
            exit 1
        fi

        # Проверяем подключение к базе данных
        if [ -z "$DB_DSN" ]; then
            error "DB_DSN не установлена в .env файле"
            exit 1
        fi

        # Выполняем миграции
        log "Выполнение миграций..."
        ./scripts/migrate.sh up

        # Собираем приложение
        log "Сборка приложения..."
        ./scripts/build.sh prod

        # Запускаем приложение
        log "Запуск приложения..."
        ./bin/gemfactory
        ;;

    docker)
        log "Развертывание в Docker..."

        # Проверяем наличие Docker
        if ! command -v docker &> /dev/null; then
            error "Docker не установлен"
            echo "Установите Docker: https://docs.docker.com/get-docker/"
            exit 1
        fi

        # Проверяем наличие Docker Compose
        if ! command -v docker-compose &> /dev/null; then
            error "Docker Compose не установлен"
            echo "Установите Docker Compose: https://docs.docker.com/compose/install/"
            exit 1
        fi

        # Собираем Docker образ
        log "Сборка Docker образа..."
        docker build -t gemfactory:latest .

        # Запускаем с Docker Compose
        log "Запуск с Docker Compose..."
        docker-compose up -d

        # Показываем статус
        log "Статус контейнеров:"
        docker-compose ps

        log "Логи приложения:"
        docker-compose logs -f app
        ;;

    production)
        log "Продакшен развертывание..."

        # Проверяем наличие необходимых переменных
        required_vars=("DB_DSN" "BOT_TOKEN" "ADMIN_USERNAME" "SPOTIFY_CLIENT_ID" "SPOTIFY_CLIENT_SECRET" "PLAYLIST_URL")
        for var in "${required_vars[@]}"; do
            if [ -z "${!var}" ]; then
                error "Переменная $var не установлена в .env файле"
                exit 1
            fi
        done

        # Собираем приложение для продакшена
        log "Сборка приложения для продакшена..."
        ./scripts/build.sh prod

        # Создаем systemd сервис (если на Linux)
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            log "Создание systemd сервиса..."
            sudo tee /etc/systemd/system/gemfactory.service > /dev/null <<EOF
[Unit]
Description=GemFactory Telegram Bot
After=network.target

[Service]
Type=simple
User=gemfactory
WorkingDirectory=$(pwd)
ExecStart=$(pwd)/bin/gemfactory
Restart=always
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

            # Перезагружаем systemd
            sudo systemctl daemon-reload
            sudo systemctl enable gemfactory
            sudo systemctl start gemfactory

            log "Сервис запущен. Статус:"
            sudo systemctl status gemfactory
        else
            warning "Автоматическое создание сервиса поддерживается только на Linux"
            info "Запустите приложение вручную: ./bin/gemfactory"
        fi
        ;;

    *)
        error "Неизвестный режим развертывания: $DEPLOY_MODE"
        echo "Использование: $0 [local|docker|production]"
        echo ""
        echo "Режимы развертывания:"
        echo "  local      - Локальное развертывание (по умолчанию)"
        echo "  docker     - Развертывание в Docker"
        echo "  production - Продакшен развертывание"
        exit 1
        ;;
esac

log "Развертывание завершено успешно!"
