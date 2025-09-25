FROM golang:1.24-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gemfactory cmd/bot/main.go

# Финальный образ
FROM alpine:3.18

# Устанавливаем необходимые пакеты
RUN apk add --no-cache ca-certificates tzdata postgresql-client

# Создаем пользователя
RUN adduser -D -s /bin/sh appuser

# Копируем бинарник
COPY --from=builder /app/gemfactory /app/gemfactory

# Копируем миграции
COPY --from=builder /app/migrations /app/migrations

# Копируем скрипт запуска
COPY deployments/start.sh /app/start.sh

# Создаем папки для данных и логов
RUN mkdir -p /app/data /app/logs

# Устанавливаем права
RUN chmod +x /app/start.sh /app/gemfactory && \
    chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Устанавливаем рабочую директорию
WORKDIR /app

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["/app/start.sh"]