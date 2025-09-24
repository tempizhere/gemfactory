#!/bin/bash

# Скрипт для развертывания приложения с внешней базой данных
# Использование: ./scripts/deploy.sh [build|up|down|logs|status]

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

# Определяем действие
ACTION=${1:-up}

# Проверяем наличие .env файла
if [ ! -f ".env" ]; then
    error ".env файл не найден"
    echo "Создайте .env файл на основе .env.example:"
    echo "cp .env.example .env"
    echo "Отредактируйте .env файл с вашими настройками"
    exit 1
fi

# Загружаем переменные окружения
source .env

# Проверяем необходимые переменные
required_vars=("DB_DSN" "BOT_TOKEN" "ADMIN_USERNAME")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        error "Переменная $var не установлена в .env файле"
        exit 1
    fi
done

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

# Определяем путь к docker-compose файлу
COMPOSE_FILE="deployments/docker-compose.yml"

case $ACTION in
    build)
        log "Сборка Docker образа..."
        docker-compose -f $COMPOSE_FILE build --no-cache
        log "Сборка завершена"
        ;;
    up)
        log "Запуск приложения с внешней базой данных..."
        docker-compose -f $COMPOSE_FILE up -d
        log "Приложение запущено"
        info "Проверка статуса:"
        docker-compose -f $COMPOSE_FILE ps
        ;;
    down)
        log "Остановка приложения..."
        docker-compose -f $COMPOSE_FILE down
        log "Приложение остановлено"
        ;;
    logs)
        log "Показ логов приложения..."
        docker-compose -f $COMPOSE_FILE logs -f app
        ;;
    status)
        log "Статус контейнеров:"
        docker-compose -f $COMPOSE_FILE ps
        ;;
    restart)
        log "Перезапуск приложения..."
        docker-compose -f $COMPOSE_FILE restart app
        log "Приложение перезапущено"
        ;;
    migrate)
        log "Применение миграций..."
        docker-compose -f $COMPOSE_FILE exec app migrate -path /app/migrations -database "$DB_DSN" up
        log "Миграции применены"
        ;;
    shell)
        log "Подключение к контейнеру..."
        docker-compose -f $COMPOSE_FILE exec app sh
        ;;
    *)
        echo "Использование: $0 [build|up|down|logs|status|restart|migrate|shell]"
        echo ""
        echo "Команды:"
        echo "  build    - Собрать Docker образ"
        echo "  up       - Запустить приложение (по умолчанию)"
        echo "  down     - Остановить приложение"
        echo "  logs     - Показать логи приложения"
        echo "  status   - Показать статус контейнеров"
        echo "  restart  - Перезапустить приложение"
        echo "  migrate  - Применить миграции вручную"
        echo "  shell    - Подключиться к контейнеру"
        echo ""
        echo "Примеры:"
        echo "  $0 build"
        echo "  $0 up"
        echo "  $0 logs"
        echo "  $0 migrate"
        exit 1
        ;;
esac



