# Makefile для GemFactory

# Переменные
BINARY_NAME=gemfactory
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=cmd/bot/main.go
MIGRATIONS_PATH=migrations
DOCKER_IMAGE=gemfactory:latest

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: help build run test clean migrate docker-build docker-run docker-stop

# Показать справку
help:
	@echo "$(GREEN)GemFactory - Makefile команды$(NC)"
	@echo ""
	@echo "$(YELLOW)Основные команды:$(NC)"
	@echo "  make build          - Собрать приложение"
	@echo "  make run            - Запустить приложение"
	@echo "  make test           - Запустить тесты"
	@echo "  make clean          - Очистить собранные файлы"
	@echo ""
	@echo "$(YELLOW)База данных:$(NC)"
	@echo "  make migrate-up     - Выполнить миграции вверх"
	@echo "  make migrate-down   - Откатить миграции"
	@echo "  make migrate-status - Показать статус миграций"
	@echo "  make migrate-create - Создать новую миграцию"
	@echo ""
	@echo "$(YELLOW)Docker:$(NC)"
	@echo "  make docker-build   - Собрать Docker образ"
	@echo "  make docker-run     - Запустить в Docker"
	@echo "  make docker-stop    - Остановить Docker контейнеры"
	@echo ""
	@echo "$(YELLOW)Разработка:$(NC)"
	@echo "  make dev            - Запустить в режиме разработки"
	@echo "  make fmt            - Форматировать код"
	@echo "  make vet            - Проверить код"
	@echo "  make lint           - Запустить линтер"

# Сборка приложения
build:
	@echo "$(GREEN)Сборка приложения...$(NC)"
	@mkdir -p bin
	@go build -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "$(GREEN)Сборка завершена: $(BINARY_PATH)$(NC)"

# Сборка для продакшена
build-prod:
	@echo "$(GREEN)Сборка для продакшена...$(NC)"
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "$(GREEN)Сборка завершена: $(BINARY_PATH)$(NC)"

# Запуск приложения
run: build
	@echo "$(GREEN)Запуск приложения...$(NC)"
	@./$(BINARY_PATH)

# Запуск в режиме разработки
dev:
	@echo "$(GREEN)Запуск в режиме разработки...$(NC)"
	@go run $(MAIN_PATH)

# Тесты
test:
	@echo "$(GREEN)Запуск тестов...$(NC)"
	@go test -v ./...

# Форматирование кода
fmt:
	@echo "$(GREEN)Форматирование кода...$(NC)"
	@go fmt ./...

# Проверка кода
vet:
	@echo "$(GREEN)Проверка кода...$(NC)"
	@go vet ./...

# Линтер
lint:
	@echo "$(GREEN)Запуск линтера...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)golangci-lint не установлен, пропускаем$(NC)"; \
	fi

# Очистка
clean:
	@echo "$(GREEN)Очистка...$(NC)"
	@rm -rf bin/
	@go clean

# Миграции вверх
migrate-up:
	@echo "$(GREEN)Выполнение миграций вверх...$(NC)"
	@./scripts/migrate.sh up

# Миграции вниз
migrate-down:
	@echo "$(GREEN)Откат миграций...$(NC)"
	@./scripts/migrate.sh down

# Статус миграций
migrate-status:
	@echo "$(GREEN)Статус миграций...$(NC)"
	@./scripts/migrate.sh status

# Создание миграции
migrate-create:
	@echo "$(GREEN)Создание новой миграции...$(NC)"
	@./scripts/migrate.sh create $(NAME)

# Сборка Docker образа
docker-build:
	@echo "$(GREEN)Сборка Docker образа...$(NC)"
	@docker build -t $(DOCKER_IMAGE) -f deployments/Dockerfile .

# Запуск в Docker
docker-run:
	@echo "$(GREEN)Запуск в Docker...$(NC)"
	@docker-compose -f deployments/docker-compose.yml up -d

# Остановка Docker контейнеров
docker-stop:
	@echo "$(GREEN)Остановка Docker контейнеров...$(NC)"
	@docker-compose -f deployments/docker-compose.yml down

# Запуск в Docker для разработки
docker-dev:
	@echo "$(GREEN)Запуск в Docker для разработки...$(NC)"
	@docker-compose -f deployments/docker-compose.dev.yml up -d

# Остановка Docker контейнеров для разработки
docker-dev-stop:
	@echo "$(GREEN)Остановка Docker контейнеров для разработки...$(NC)"
	@docker-compose -f deployments/docker-compose.dev.yml down

# Установка зависимостей
deps:
	@echo "$(GREEN)Установка зависимостей...$(NC)"
	@go mod download
	@go mod tidy

# Проверка всех зависимостей
check: fmt vet test
	@echo "$(GREEN)Все проверки пройдены!$(NC)"

# Полная сборка и проверка
all: clean deps check build
	@echo "$(GREEN)Полная сборка завершена!$(NC)"

# Показать информацию о проекте
info:
	@echo "$(GREEN)Информация о проекте:$(NC)"
	@echo "  Название: GemFactory"
	@echo "  Версия Go: $(shell go version)"
	@echo "  Путь к бинарному файлу: $(BINARY_PATH)"
	@echo "  Путь к миграциям: $(MIGRATIONS_PATH)"
	@echo "  Docker образ: $(DOCKER_IMAGE)"
