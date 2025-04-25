# Этап сборки
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gemfactory cmd/bot/main.go

# Финальный образ
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gemfactory .
# Копируем начальные файлы вайтлистов в образ
COPY internal/features/releasesbot/data/ /app/internal/features/releasesbot/data/

# Создаём entrypoint скрипт для инициализации вайтлистов
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Запускаем через entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]