#!/bin/bash

# Скрипт для сборки приложения
# Использование: ./scripts/build.sh [dev|prod]

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

# Определяем режим сборки
BUILD_MODE=${1:-dev}

# Создаем директорию для бинарных файлов
mkdir -p bin

# Очищаем предыдущую сборку
log "Очистка предыдущей сборки..."
rm -f bin/gemfactory
rm -f bin/gemfactory.exe

# Проверяем наличие go.mod
if [ ! -f "go.mod" ]; then
    error "go.mod не найден. Запустите скрипт из корневой директории проекта"
    exit 1
fi

# Загружаем зависимости
log "Загрузка зависимостей..."
go mod download
go mod tidy

# Проверяем код
log "Проверка кода..."
go vet ./...
go fmt ./...

# Запускаем тесты (если есть)
if [ -d "test" ] || find . -name "*_test.go" | grep -q .; then
    log "Запуск тестов..."
    go test ./...
else
    warning "Тесты не найдены, пропускаем"
fi

# Определяем переменные сборки
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Флаги сборки
LDFLAGS="-X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT -X main.Version=$VERSION"

case $BUILD_MODE in
    dev)
        log "Сборка в режиме разработки..."
        CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o bin/gemfactory cmd/bot/main.go
        log "Сборка завершена: bin/gemfactory"
        ;;
    prod)
        log "Сборка в режиме продакшена..."
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS -s -w" -o bin/gemfactory cmd/bot/main.go
        log "Сборка завершена: bin/gemfactory (Linux amd64)"
        ;;
    windows)
        log "Сборка для Windows..."
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS -s -w" -o bin/gemfactory.exe cmd/bot/main.go
        log "Сборка завершена: bin/gemfactory.exe (Windows amd64)"
        ;;
    *)
        error "Неизвестный режим сборки: $BUILD_MODE"
        echo "Использование: $0 [dev|prod|windows]"
        echo ""
        echo "Режимы сборки:"
        echo "  dev      - Сборка для разработки (по умолчанию)"
        echo "  prod     - Сборка для продакшена (Linux amd64)"
        echo "  windows  - Сборка для Windows (amd64)"
        exit 1
        ;;
esac

# Показываем информацию о собранном файле
if [ -f "bin/gemfactory" ]; then
    info "Размер файла: $(du -h bin/gemfactory | cut -f1)"
    info "Версия: $VERSION"
    info "Коммит: $GIT_COMMIT"
    info "Время сборки: $BUILD_TIME"
elif [ -f "bin/gemfactory.exe" ]; then
    info "Размер файла: $(du -h bin/gemfactory.exe | cut -f1)"
    info "Версия: $VERSION"
    info "Коммит: $GIT_COMMIT"
    info "Время сборки: $BUILD_TIME"
fi

log "Сборка завершена успешно!"
