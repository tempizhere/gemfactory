# Makefile для GemFactory

# Переменные
BINARY_NAME=gemfactory
BUILD_DIR=build
DOCKER_IMAGE=tempizhere/gemfactory
DOCKER_TAG=latest

# Go переменные
GO=go
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: help build test clean docker-build docker-push docker-run dev run lint format

# Помощь
help: ## Показать справку
	@echo "$(GREEN)Доступные команды:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

# Сборка
build: ## Собрать бинарный файл
	@echo "$(GREEN)Сборка $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/bot/main.go
	@echo "$(GREEN)Сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-linux: ## Собрать для Linux
	@echo "$(GREEN)Сборка для Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/bot/main.go
	@echo "$(GREEN)Сборка для Linux завершена$(NC)"

# Тестирование
test: ## Запустить тесты
	@echo "$(GREEN)Запуск тестов...$(NC)"
	$(GO) test -v ./...

test-coverage: ## Запустить тесты с покрытием
	@echo "$(GREEN)Запуск тестов с покрытием...$(NC)"
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Отчет о покрытии сохранен в coverage.html$(NC)"

# Линтинг и форматирование
lint: ## Запустить линтер
	@echo "$(GREEN)Проверка кода...$(NC)"
	golangci-lint run

format: ## Форматировать код
	@echo "$(GREEN)Форматирование кода...$(NC)"
	$(GO) fmt ./...
	$(GO) vet ./...

# Очистка
clean: ## Очистить артефакты сборки
	@echo "$(GREEN)Очистка...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Очистка завершена$(NC)"

# Docker
docker-build: ## Собрать Docker образ
	@echo "$(GREEN)Сборка Docker образа...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)Docker образ собран: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

docker-push: ## Отправить Docker образ в registry
	@echo "$(GREEN)Отправка Docker образа...$(NC)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Docker образ отправлен$(NC)"

docker-run: ## Запустить в Docker
	@echo "$(GREEN)Запуск в Docker...$(NC)"
	docker-compose up -d

docker-stop: ## Остановить Docker контейнер
	@echo "$(GREEN)Остановка Docker контейнера...$(NC)"
	docker-compose down

# Разработка
dev: ## Запустить в режиме разработки
	@echo "$(GREEN)Запуск в режиме разработки...$(NC)"
	$(GO) run cmd/bot/main.go

run: build ## Собрать и запустить
	@echo "$(GREEN)Запуск приложения...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME)

# Зависимости
deps: ## Установить зависимости
	@echo "$(GREEN)Установка зависимостей...$(NC)"
	$(GO) mod download
	$(GO) mod tidy

# Проверка безопасности
security: ## Проверить зависимости на уязвимости
	@echo "$(GREEN)Проверка безопасности...$(NC)"
	$(GO) list -json -deps ./... | nancy sleuth

# Производительность
benchmark: ## Запустить бенчмарки
	@echo "$(GREEN)Запуск бенчмарков...$(NC)"
	$(GO) test -bench=. -benchmem ./...

performance-test: ## Тест производительности
	@echo "$(GREEN)Тест производительности...$(NC)"
	$(GO) test -bench=. -benchtime=5s -count=3 ./...

# Полная сборка и тестирование
all: clean deps format lint test build ## Полная сборка и тестирование
	@echo "$(GREEN)Все этапы завершены успешно!$(NC)"

# CI/CD
ci: deps lint test build-linux ## Команды для CI/CD
	@echo "$(GREEN)CI/CD pipeline завершен$(NC)"

# Деплой
deploy-staging: ## Деплой на staging
	@echo "$(GREEN)Деплой на staging...$(NC)"
	docker-compose -f docker-compose.staging.yml up -d

deploy-production: ## Деплой на production
	@echo "$(GREEN)Деплой на production...$(NC)"
	docker-compose -f docker-compose.production.yml up -d

# Мониторинг
logs: ## Показать логи
	@echo "$(GREEN)Показать логи...$(NC)"
	docker-compose logs -f

status: ## Статус сервисов
	@echo "$(GREEN)Статус сервисов...$(NC)"
	docker-compose ps